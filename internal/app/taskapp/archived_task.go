package taskapp

import (
	"context"
	"errors"
	"fmt"

	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/ctxlogger"
)

type CacheManager interface {
	FindAllConsumers(ctx context.Context) ([]domain.Event, error)
	FindAllQueues(ctx context.Context) ([]Queue, error)
	FindArchivedTasks(ctx context.Context, queue string) ([]string, error)
	GetMsgArchivedTask(ctx context.Context, queue, task string) (RawMsg, error)
	RemoveMsgArchivedTask(ctx context.Context, queue, task string) error
	RemoveItemsArchivedTasks(ctx context.Context, queue string, tasks ...string) error
	SetArchivedTasks(ctx context.Context, events []domain.Event) error
}

type Storer interface {
	GetAllSchedulers(ctx context.Context, state string) ([]domain.Event, error)
}

type TaskManager struct {
	store Storer
	cm    CacheManager
}

func NewTaskManager(store Storer, cm CacheManager) *TaskManager {
	return &TaskManager{store: store, cm: cm}
}

func (n TaskManager) GetMsgsArchived(ctx context.Context) ([]TaskArchivedData, error) {
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
