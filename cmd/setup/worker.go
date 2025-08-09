package setup

import (
	"github.com/IsaacDSC/webhook/internal/cfg"
	"github.com/IsaacDSC/webhook/pkg/asynqsvc"
	"log"

	"github.com/IsaacDSC/webhook/internal/eventqueue"
	"github.com/IsaacDSC/webhook/pkg/cache"
	"github.com/IsaacDSC/webhook/pkg/publisher"

	"github.com/IsaacDSC/webhook/internal/interstore"

	"github.com/hibiken/asynq"
)

func StartWorker(cache cache.Cache, store interstore.Repository, pub publisher.Publisher) {
	cfg := cfg.Get()

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.Cache.CacheAddr},
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)

	mux := asynq.NewServeMux()
	mux.Use(AsynqLogger)

	events := []asynqsvc.AsynqHandle{
		eventqueue.GetRequestHandle(),
		eventqueue.GetInternalConsumerHandle(store, cache, pub),
	}

	for _, event := range events {
		mux.HandleFunc(event.Event, event.Handler)
	}

	if err := srv.Run(mux); err != nil {
		log.Fatalf("could not run server: %v", err)
	}
}
