package setup

import (
	"github.com/IsaacDSC/gopherline/internal/cfg"
	"github.com/IsaacDSC/gopherline/pkg/asynqsvc"
	"log"

	"github.com/IsaacDSC/gopherline/internal/eventqueue"
	"github.com/IsaacDSC/gopherline/pkg/cache"
	"github.com/IsaacDSC/gopherline/pkg/publisher"

	"github.com/IsaacDSC/gopherline/internal/interstore"

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
