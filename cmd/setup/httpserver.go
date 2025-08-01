package setup

import (
	"log"
	"net/http"

	"github.com/IsaacDSC/webhook/internal/infra/cfg"
	"github.com/IsaacDSC/webhook/internal/infra/handler"
	"github.com/IsaacDSC/webhook/internal/infra/middleware"
	"github.com/IsaacDSC/webhook/internal/interstore"
	"github.com/IsaacDSC/webhook/internal/intersvc"
	cache2 "github.com/IsaacDSC/webhook/pkg/cache"
	"github.com/IsaacDSC/webhook/pkg/publisher"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func StartServer(mongodb *mongo.Client, redisClient *redis.Client) {
	cfg := cfg.Get()

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.Cache.CacheAddr})
	defer asynqClient.Close()

	cache := cache2.NewStrategy(redisClient)
	store := interstore.NewMongoStore(mongodb)
	svc := intersvc.NewWeb(store, cache)
	pub := publisher.NewPublisher(asynqClient)
	handlers := handler.NewHandler(svc, pub)

	mux := http.NewServeMux()
	for p, h := range handlers.GetRoutes() {
		mux.HandleFunc(p, h)
	}

	handler := middleware.LoggerMiddleware(mux)

	log.Println("Starting HTTP server on :8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		panic(err)
	}
}
