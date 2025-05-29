package setup

import (
	"context"
	cache2 "github.com/IsaacDSC/webhook/internal/infra/cache"
	"github.com/IsaacDSC/webhook/internal/infra/cfg"
	"github.com/IsaacDSC/webhook/internal/infra/handler"
	"github.com/IsaacDSC/webhook/internal/infra/repository"
	"github.com/IsaacDSC/webhook/internal/service"
	"github.com/IsaacDSC/webhook/pkg/publisher"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"log"
	"net/http"
)

func StartServer(repository *repository.MongoRepo) {
	ctx := context.Background()
	cfg := cfg.Get()

	cacheClient := redis.NewClient(&redis.Options{Addr: cfg.Cache.CacheAddr})
	if err := cacheClient.Ping(ctx).Err(); err != nil {
		panic(err)
	}

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.Cache.CacheAddr})
	defer asynqClient.Close()

	pub := publisher.NewPublisher(asynqClient)
	cache := cache2.NewStrategy(cacheClient)
	svc := service.NewService(repository, pub, cache)
	handlers := handler.NewHandler(svc)

	mux := http.NewServeMux()
	for p, h := range handlers.GetRoutes() {
		mux.HandleFunc(p, h)
	}

	log.Println("Starting HTTP server on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}
}
