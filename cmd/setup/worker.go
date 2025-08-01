package setup

import (
	"log"

	"github.com/IsaacDSC/webhook/internal/consworker"
	"github.com/IsaacDSC/webhook/internal/infra/cfg"
	"github.com/IsaacDSC/webhook/internal/infra/middleware"
	"github.com/IsaacDSC/webhook/internal/interstore"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/hibiken/asynq"
)

func StartWorker(mongodb *mongo.Client) {
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

	store := interstore.NewMongoStore(mongodb)
	mux := asynq.NewServeMux()
	mux.Use(middleware.AsynqLogger)
	tasks := map[consworker.TaskName]asynq.HandlerFunc{
		consworker.PublisherExternalEvent: consworker.GetInternalConsumerHandle(store.GetInternalEvent),
	}

	for e, h := range tasks {
		mux.HandleFunc(e.String(), h)
	}

	if err := srv.Run(mux); err != nil {
		log.Fatalf("could not run server: %v", err)
	}
}
