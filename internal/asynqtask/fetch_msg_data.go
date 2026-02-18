package asynqtask

import "github.com/IsaacDSC/gqueue/internal/domain"

type FetchMsg struct {
	ID         string
	QueueName  string
	Tasks      []string
	Data       any
	Schedulers []domain.Consumer
}

type TaskArchivedData struct {
	ID        string         `json:"id"`
	Queue     string         `json:"queue"`
	Consumers []Consumer     `json:"consumers,omitempty"`
	Msg       map[string]any `json:"msg"`
	Tasks     []string       `json:"tasks"`
}
