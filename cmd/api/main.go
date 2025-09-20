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
)

const appName = "gqueue"

// go run . --service=server
// go run . --service=worker
// go run . --service=archived-notification
// go run . [server, worker]
func main() {
	cfg := cfg.Get()
	ctx := context.Background()

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.Cache.CacheAddr})
	defer asynqClient.Close()

	cacheClient := redis.NewClient(&redis.Options{Addr: cfg.Cache.CacheAddr})
	if err := cacheClient.Ping(ctx).Err(); err != nil {
		panic(err)
	}

	store, err := interstore.NewPostgresStoreFromDSN(cfg.ConfigDatabase.DbConn)
	if err != nil {
		panic(err)
	}

	cc := cachemanager.NewStrategy(appName, cacheClient)
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
		setup.StartArchivedNotify(ctx, store, cacheClient)
		return
	}

	go setup.StartServer(cacheClient, cc, store, pub)
	setup.StartWorker(cc, store, pub)

}
