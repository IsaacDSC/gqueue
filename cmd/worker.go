package cmd

import (
	"github.com/IsaacDSC/webhook/internal/infra/cfg"
	"github.com/IsaacDSC/webhook/internal/infra/gateway"
	"github.com/IsaacDSC/webhook/internal/infra/repository"
	"github.com/IsaacDSC/webhook/internal/infra/task"
	"log"

	"github.com/hibiken/asynq"
)

func StartWorker(repository *repository.MongoRepo) {
	cfg := cfg.Get()

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.Cache.CacheAddr},
		asynq.Config{
			// Specify how many concurrent workers to use
			Concurrency: 10,
			// Optionally specify multiple queues with different priority.
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
			// See the godoc for other configuration options
		},
	)

	gate := gateway.NewHook()
	tasks := task.NewTasks(repository, gate)

	mux := asynq.NewServeMux()
	mux.Use(task.LogMiddleware)
	for e, h := range tasks.GetTasks() {
		mux.HandleFunc(e.String(), h)
	}

	if err := srv.Run(mux); err != nil {
		log.Fatalf("could not run server: %v", err)
	}
}
