package main

import (
	"context"
	"flag"

	"github.com/IsaacDSC/webhook/cmd/setup"
	"github.com/IsaacDSC/webhook/internal/infra/cfg"
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

	cacheClient := redis.NewClient(&redis.Options{Addr: cfg.Cache.CacheAddr})
	if err := cacheClient.Ping(ctx).Err(); err != nil {
		panic(err)
	}

	service := flag.String("service", "all", "service to run")
	flag.Parse()

	if *service == "worker" {
		setup.StartWorker(mongodb)
		return
	}

	if *service == "webhook" {
		setup.StartServer(mongodb, cacheClient)
		return
	}

	go setup.StartServer(mongodb, cacheClient)
	setup.StartWorker(mongodb)

}
