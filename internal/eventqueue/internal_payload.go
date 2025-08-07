package eventqueue

import (
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
)

type InternalPayload struct {
	EventName string     `json:"event_name"`
	Data      Data       `json:"data"`
	Metadata  Metadata   `json:"metadata"`
	Opts      ConfigOpts `json:"opts"`
}

func (p InternalPayload) GetOpts() []asynq.Option {
	return p.Opts.ToAsynqOptions()
}

type Metadata struct {
	Source      string            `json:"source"`
	Version     string            `json:"version"`
	Environment string            `json:"environment"`
	Headers     map[string]string `json:"headers"`
}

type ConfigOpts struct {
	MaxRetries uint          `json:"max_retries"`
	Retention  time.Duration `json:"retention"`
	Deadline   *time.Time    `json:"deadline"`
	UniqueTtl  time.Duration `json:"unique_ttl"`
	ScheduleIn time.Duration `json:"schedule_in"`
	Queue      string        `json:"queue"`
}

func (co ConfigOpts) ToAsynqOptions() []asynq.Option {
	opts := []asynq.Option{}

	if co.MaxRetries > 0 {
		opts = append(opts, asynq.MaxRetry(int(co.MaxRetries)))
	}
	if co.Retention > 0 {
		opts = append(opts, asynq.Retention(co.Retention*time.Second))
	}
	if co.Deadline != nil {
		opts = append(opts, asynq.Deadline(*co.Deadline))
	}
	if co.UniqueTtl > 0 {
		opts = append(opts, asynq.Unique(co.UniqueTtl*time.Second))
	}
	if co.ScheduleIn > 0 {
		opts = append(opts, asynq.ProcessIn(co.ScheduleIn*time.Second))
	}
	if co.Queue != "" {
		opts = append(opts, asynq.Queue(co.Queue))
	}

	return opts
}

type Data map[string]any

func (d Data) ToBytes() []byte {
	data, _ := json.Marshal(d)
	return data
}
