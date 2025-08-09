package setup

import (
	"github.com/IsaacDSC/gqueue/internal/cfg"
	"github.com/IsaacDSC/gqueue/pkg/asynqsvc"
	"log"

	"github.com/IsaacDSC/gqueue/internal/eventqueue"
	"github.com/IsaacDSC/gqueue/pkg/cache"
	"github.com/IsaacDSC/gqueue/pkg/publisher"

	"github.com/IsaacDSC/gqueue/internal/interstore"

	"github.com/hibiken/asynq"
)

func StartWorker(cache cache.Cache, store interstore.Repository, pub publisher.Publisher) {
	cfg := cfg.Get()

	asyqCfg := asynq.Config{
		Concurrency: cfg.AsynqConfig.Concurrency,
		Queues:      cfg.AsynqConfig.Queues,
	}
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.Cache.CacheAddr},
		asyqCfg,
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

	log.Println("[*] starting worker with configs")
	log.Println("[*] wq.concurrency", asyqCfg.Concurrency)
	log.Println("[*] wq.queues", asyqCfg.Queues)

	if err := srv.Run(mux); err != nil {
		log.Fatalf("could not run server: %v", err)
	}
}
