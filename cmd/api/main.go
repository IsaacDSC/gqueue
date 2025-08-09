package main

import (
	"context"
	"flag"
	"github.com/IsaacDSC/gqueue/internal/cfg"

	"github.com/hibiken/asynq"

	"github.com/IsaacDSC/gqueue/cmd/setup"
	"github.com/IsaacDSC/gqueue/internal/interstore"
	"github.com/IsaacDSC/gqueue/pkg/cache"
	"github.com/IsaacDSC/gqueue/pkg/publisher"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// go run . --service=worker
// go run . --service=webhook
// go run . --service=all
func main() {
	cfg := cfg.Get()
	ctx := context.Background()

	mongodb, err := mongo.Connect(options.Client().ApplyURI(cfg.ConfigDatabase.DbConn))
	if err != nil {
		panic(err)
	}

	defer func() {
		if err = mongodb.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	if err := mongodb.Ping(ctx, nil); err != nil {
		panic(err)
	}

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.Cache.CacheAddr})
	defer asynqClient.Close()

	cacheClient := redis.NewClient(&redis.Options{Addr: cfg.Cache.CacheAddr})
	if err := cacheClient.Ping(ctx).Err(); err != nil {
		panic(err)
	}

	cc := cache.NewStrategy(cacheClient)
	store := interstore.NewMongoStore(mongodb)
	pub := publisher.NewPublisher(asynqClient)

	service := flag.String("service", "all", "service to run")
	flag.Parse()

	if *service == "worker" {
		setup.StartWorker(cc, store, pub)
		return
	}

	if *service == "webhook" {
		setup.StartServer(cc, store, pub)
		return
	}

	go setup.StartServer(cc, store, pub)
	setup.StartWorker(cc, store, pub)

}
