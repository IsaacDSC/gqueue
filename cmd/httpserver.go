package cmd

import (
	"github.com/IsaacDSC/webhook/internal/infra/cfg"
	"github.com/IsaacDSC/webhook/internal/infra/handler"
	"github.com/IsaacDSC/webhook/internal/infra/repository"
	"github.com/IsaacDSC/webhook/internal/service"
	"github.com/IsaacDSC/webhook/pkg/publisher"
	"github.com/hibiken/asynq"
	"log"
	"net/http"
)

func StartServer(repository *repository.MongoRepo) {
	cfg := cfg.Get()
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.Cache.CacheAddr})
	defer client.Close()

	pub := publisher.NewPublisher(client)
	svc := service.NewService(repository, pub)
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
