package main

import (
	"context"
	"flag"

	"github.com/IsaacDSC/gqueue/internal/cfg"

	"github.com/hibiken/asynq"

	"github.com/IsaacDSC/gqueue/cmd/setup"
	"github.com/IsaacDSC/gqueue/internal/interstore"
	"github.com/IsaacDSC/gqueue/pkg/cachemanager"
	"github.com/IsaacDSC/gqueue/pkg/publisher"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// go run . --service=server
// go run . --service=webhook
// go run . --service=archived-notification
// go run . --service=all
func main() {
	cfg := cfg.Get()
	ctx := context.Background()

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.Cache.CacheAddr})
	defer asynqClient.Close()

	cacheClient := redis.NewClient(&redis.Options{Addr: cfg.Cache.CacheAddr})
	if err := cacheClient.Ping(ctx).Err(); err != nil {
		panic(err)
	}

	var store interstore.Repository
	if cfg.ConfigDatabase.Driver == "pg" {
		conn, err := interstore.NewPostgresStoreFromDSN(cfg.ConfigDatabase.DbConn)
		if err != nil {
			panic(err)
		}

		store = conn
	} else {
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

		store = interstore.NewMongoStore(mongodb)
	}

	cc := cachemanager.NewStrategy(cacheClient)
	pub := publisher.NewPublisher(asynqClient)

	service := flag.String("service", "all", "service to run")
	flag.Parse()

	if *service == "worker" {
		setup.StartWorker(cc, store, pub)
		return
	}

	// TODO: adicionar graceful shutdown
	if *service == "server" {
		setup.StartServer(cacheClient, cc, store, pub)
		return
	}

	// TODO: adicionar graceful shutdown
	if *service == "archived-notification" {
		setup.StartArchivedNotify(ctx, cacheClient)
		return
	}

	go setup.StartArchivedNotify(ctx, cacheClient)
	go setup.StartServer(cacheClient, cc, store, pub)
	setup.StartWorker(cc, store, pub)

}
