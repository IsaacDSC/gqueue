package task

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/IsaacDSC/gqueue/pkg/ctxlogger"
)

type TaskArchivedData struct {
	ID        string         `json:"id"`
	Queue     string         `json:"queue"`
	Consumers []Consumer     `json:"consumers,omitempty"`
	Msg       map[string]any `json:"msg"`
	Tasks     []string       `json:"tasks"`
}

type ArchivedStorer interface {
	FindAllConsumers(ctx context.Context) (QueueConsumers, error)
	FindAllQueues(ctx context.Context) (Queues, error)
	FindArchivedTasks(ctx context.Context, queue string) ([]string, error)
	GetMsgArchivedTask(ctx context.Context, queue, task string) (RawMsg, error)
	RemoveMsgArchivedTask(ctx context.Context, queue, task string) error
	RemoveItemsArchivedTasks(ctx context.Context, queue string, tasks ...string) error
}

type Fetcher interface {
	NotifyConsumer(ctx context.Context, url string, data map[string]any, headers map[string]string) error
}

type TaskManager struct {
	store ArchivedStorer
	fetch Fetcher
}

func NewTaskManager(store ArchivedStorer, fetch Fetcher) *TaskManager {
	return &TaskManager{store: store, fetch: fetch}
}

func (n TaskManager) NotifyListeners(ctx context.Context) error {
	l := ctxlogger.GetLogger(ctx)
	l.Debug("Starting notification process")

	msgs, err := n.getMsgsArchived(ctx)
	if err != nil {
		l.Error("Failed to get archived messages", "error", err)
		return fmt.Errorf("failed to get archived messages: %w", err)
	}

	l.Debug("Fetched archived messages", "count", len(msgs))

	var wg sync.WaitGroup
	wg.Add(len(msgs))

	for _, msg := range msgs {
		for _, consumer := range msg.Consumers {
			go func() {
				if err := n.fetch.NotifyConsumer(ctx, consumer.Host, msg.Msg, consumer.Headers); err != nil {
					l.Error("Failed to notify consumer", "consumer", consumer.Host, "error", err)
				} else {
					if err := n.store.RemoveMsgArchivedTask(ctx, msg.Queue, msg.ID); err != nil {
						l.Error("Failed to remove archived task", "queue", msg.Queue, "task", msg.ID, "error", err)
					} else {
						if err := n.store.RemoveItemsArchivedTasks(ctx, msg.Queue, msg.Tasks...); err != nil {
							l.Error("Failed to remove archived tasks", "queue", msg.Queue, "tasks", msg.Tasks, "error", err)
						}
					}
				}
			}()

		}

		l.Info("Notifying consumers", "queue", msg.Queue, "consumers", msg.Consumers)
	}

	wg.Wait()
	l.Debug("Notification process completed")

	return nil
}

func (n TaskManager) getMsgsArchived(ctx context.Context) ([]TaskArchivedData, error) {
	l := ctxlogger.GetLogger(ctx)

	queueOnConsumers, err := n.store.FindAllConsumers(ctx)

	if errors.Is(err, ErrorNotFound) {
		l.Warn("No consumers found")
		return []TaskArchivedData{}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get consumers: %w", err)
	}

	qResult, err := n.store.FindAllQueues(ctx)
	if errors.Is(err, ErrorNotFound) {
		l.Warn("No queues found")
		return []TaskArchivedData{}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get queues: %w", err)
	}

	queueOnConsumers.RmNotContains(qResult)
	archived := make(map[string][]string)
	for q := range queueOnConsumers {

		result, err := n.store.FindArchivedTasks(ctx, q.String())
		if err != nil {
			l.Error("Failed to fetch archived tasks", "queue", q, "error", err)
			continue
		}

		archived[q.String()] = result
	}

	var output []TaskArchivedData
	for queue, tasks := range archived {
		for _, task := range tasks {
			result, err := n.store.GetMsgArchivedTask(ctx, queue, task)
			if err != nil {
				l.Error("Failed to fetch archived task message", "queue", queue, "task", task, "error", err)
				continue
			}
			output = append(output, TaskArchivedData{
				ID:        task,
				Queue:     queue,
				Consumers: queueOnConsumers[Queue(queue)],
				Msg:       result.Msg(),
				Tasks:     tasks,
			})
		}
	}

	return output, nil

}

func (n TaskManager) GetAllMsgsByQueue(ctx context.Context, queue string) ([]TaskArchivedData, error) {
	l := ctxlogger.GetLogger(ctx)

	tasksResult, err := n.store.FindArchivedTasks(ctx, queue)
	if errors.Is(err, ErrorNotFound) {
		return []TaskArchivedData{}, nil
	}

	if err != nil {
		l.Error("Failed to fetch archived tasks", "queue", queue, "error", err)
		return nil, fmt.Errorf("failed to get archived tasks: %w", err)
	}

	var output []TaskArchivedData
	for _, task := range tasksResult {
		result, err := n.store.GetMsgArchivedTask(ctx, queue, task)
		if err != nil {
			l.Error("Failed to fetch archived task message", "queue", queue, "task", task, "error", err)
			continue
		}
		output = append(output, TaskArchivedData{
			ID:        task,
			Queue:     queue,
			Consumers: nil, // You might want to fetch consumers for this queue if needed
			Msg:       result.Msg(),
			Tasks:     tasksResult,
		})
	}

	return output, nil
}
