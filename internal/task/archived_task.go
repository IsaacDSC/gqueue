package task

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/ctxlogger"
)

type CacheManager interface {
	FindAllTriggers(ctx context.Context) ([]domain.Event, error)
	FindAllQueues(ctx context.Context) ([]Queue, error)
	FindArchivedTasks(ctx context.Context, queue string) ([]string, error)
	GetMsgArchivedTask(ctx context.Context, queue, task string) (RawMsg, error)
	RemoveMsgArchivedTask(ctx context.Context, queue, task string) error
	RemoveItemsArchivedTasks(ctx context.Context, queue string, tasks ...string) error
	SetArchivedTasks(ctx context.Context, events []domain.Event) error
}

type Fetcher interface {
	NotifyScheduler(ctx context.Context, url string, data any, headers map[string]string) error
}

type Storer interface {
	GetAllSchedulers(ctx context.Context, state string) ([]domain.Event, error)
}

type TaskManager struct {
	store Storer
	cm    CacheManager
	fetch Fetcher
}

func NewTaskManager(store Storer, cm CacheManager, fetch Fetcher) *TaskManager {
	return &TaskManager{store: store, cm: cm, fetch: fetch}
}

func (n TaskManager) NotifyListeners(ctx context.Context) error {
	l := ctxlogger.GetLogger(ctx)
	l.Debug("Starting notification process")

	events, err := n.cm.FindAllTriggers(ctx)
	if errors.Is(err, ErrorNotFound) {
		results, err := n.store.GetAllSchedulers(ctx, "archived")
		if errors.Is(domain.EventNotFound, err) {
			l.Warn("No consumers found", "tag", "TaskManager.NotifyListeners")
			return nil
		}

		if err != nil {
			return fmt.Errorf("failed to get consumers on store: %w", err)
		}

		if err := n.cm.SetArchivedTasks(ctx, results); err != nil {
			l.Error("Failed to set archived tasks", "error", err)
		}

		events = results
	}

	if err != nil {
		return fmt.Errorf("failed to get consumers on cache: %w", err)
	}

	msgs, err := n.getMsgsArchived(ctx)
	if err != nil {
		l.Error("Failed to get archived messages", "error", err)
		return fmt.Errorf("failed to get archived messages: %w", err)
	}

	var fetchMsgs []FetchMsg
	for _, events := range events {
		for _, msg := range msgs {
			if events.Name == msg.Msg["event_name"] {
				fetchMsgs = append(fetchMsgs, FetchMsg{
					ID:         msg.ID,
					QueueName:  msg.Queue,
					Tasks:      msg.Tasks,
					Data:       msg.Msg["data"],
					Schedulers: events.Triggers,
				})
			}
		}
	}

	l.Debug("Fetched archived messages", "count", len(msgs))

	var wg sync.WaitGroup
	for _, msg := range fetchMsgs {
		for _, scheduler := range msg.Schedulers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := n.fetch.NotifyScheduler(ctx, scheduler.Host, msg.Data, scheduler.Headers); err != nil {
					l.Error("Failed to notify consumer", "consumer", scheduler.Host, "error", err)
				} else {
					if err := n.cm.RemoveMsgArchivedTask(ctx, msg.QueueName, msg.ID); err != nil {
						l.Error("Failed to remove archived task", "queue", msg.QueueName, "task", msg.ID, "error", err)
					} else {
						if err := n.cm.RemoveItemsArchivedTasks(ctx, msg.QueueName, msg.Tasks...); err != nil {
							l.Error("Failed to remove archived tasks", "queue", msg.QueueName, "tasks", msg.Tasks, "error", err)
						}
					}
				}
			}()

		}

		l.Info("Notifying schedulers", "queue", msg.QueueName, "schedulers", msg.Schedulers)
	}

	wg.Wait()
	l.Debug("Notification process completed")

	return nil
}

func (n TaskManager) getMsgsArchived(ctx context.Context) ([]TaskArchivedData, error) {
	l := ctxlogger.GetLogger(ctx)

	queues, err := n.cm.FindAllQueues(ctx)
	if errors.Is(err, ErrorNotFound) {
		l.Warn("No queues found")
		return []TaskArchivedData{}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get queues: %w", err)
	}

	archived := make(map[string][]string)
	for _, queue := range queues {
		result, err := n.cm.FindArchivedTasks(ctx, queue.String())
		if err != nil {
			l.Error("Failed to fetch archived tasks", "queue", queue, "error", err)
			continue
		}

		archived[queue.String()] = result
	}

	var output []TaskArchivedData
	for queue, tasks := range archived {
		for _, task := range tasks {
			result, err := n.cm.GetMsgArchivedTask(ctx, queue, task)
			if err != nil {
				l.Error("Failed to fetch archived task message", "queue", queue, "task", task, "error", err)
				continue
			}
			output = append(output, TaskArchivedData{
				ID:        task,
				Queue:     queue,
				Consumers: []Consumer{},
				Msg:       result.Msg(),
				Tasks:     tasks,
			})
		}
	}

	return output, nil

}

func (n TaskManager) GetAllMsgsByQueue(ctx context.Context, queue string) ([]TaskArchivedData, error) {
	l := ctxlogger.GetLogger(ctx)

	tasksResult, err := n.cm.FindArchivedTasks(ctx, queue)
	if errors.Is(err, ErrorNotFound) {
		return []TaskArchivedData{}, nil
	}

	if err != nil {
		l.Error("Failed to fetch archived tasks", "queue", queue, "error", err)
		return nil, fmt.Errorf("failed to get archived tasks: %w", err)
	}

	var output []TaskArchivedData
	for _, task := range tasksResult {
		result, err := n.cm.GetMsgArchivedTask(ctx, queue, task)
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
