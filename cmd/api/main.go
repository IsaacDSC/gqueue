package main

import (
	"context"
	"flag"
	"log"
	"time"

	"cloud.google.com/go/pubsub"
	vkit "cloud.google.com/go/pubsub/apiv1"
	"github.com/IsaacDSC/gqueue/internal/cfg"
	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/internal/storests"
	"github.com/googleapis/gax-go/v2"
	"google.golang.org/grpc/codes"

	"github.com/hibiken/asynq"

	"github.com/IsaacDSC/gqueue/cmd/setup"
	"github.com/IsaacDSC/gqueue/internal/interstore"
	"github.com/IsaacDSC/gqueue/pkg/cachemanager"
	"github.com/IsaacDSC/gqueue/pkg/publisher"
	"github.com/redis/go-redis/v9"
)

const appName = "gqueue"

// TODO: rename to --scope=...
// go run . --service=server
// go run . --service=worker
// go run . --service=archived-notification
// go run . [server, worker]
func main() {
	cfg := cfg.Get()
	ctx := context.Background()

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.Cache.CacheAddr})
	defer asynqClient.Close()

	config := &pubsub.ClientConfig{
		PublisherCallOptions: &vkit.PublisherCallOptions{
			Publish: []gax.CallOption{
				gax.WithRetry(func() gax.Retryer {
					return gax.OnCodes([]codes.Code{
						codes.Aborted,
						codes.Canceled,
						codes.Internal,
						codes.ResourceExhausted,
						codes.Unknown,
						codes.Unavailable,
						codes.DeadlineExceeded,
					}, gax.Backoff{
						Initial:    250 * time.Millisecond, // default 100 milliseconds
						Max:        5 * time.Second,        // default 60 seconds
						Multiplier: 1.45,                   // default 1.3
					})
				}),
			},
		},
	}

	clientPubsub, err := pubsub.NewClientWithConfig(ctx, domain.ProjectID, config)
	if err != nil {
		log.Fatalf("Erro ao criar cliente: %v", err)
	}

	defer clientPubsub.Close()

	cacheClient := redis.NewClient(&redis.Options{Addr: cfg.Cache.CacheAddr})
	if err := cacheClient.Ping(ctx).Err(); err != nil {
		panic(err)
	}

	insights := storests.NewStore(cacheClient)

	store, err := interstore.NewPostgresStoreFromDSN(cfg.ConfigDatabase.DbConn)
	if err != nil {
		panic(err)
	}

	cc := cachemanager.NewStrategy(appName, cacheClient)

	var pub publisher.Publisher
	if cfg.AsynqConfig.WorkerType == "googlepubsub" {
		pub = publisher.NewPubSubGoogle(clientPubsub)
	} else {
		pub = publisher.NewPublisher(asynqClient)

	}

	service := flag.String("service", "all", "service to run")
	flag.Parse()

	if *service == "worker" {
		setup.StartWorker(clientPubsub, cc, store, pub, insights)
		return
	}

	// TODO: adicionar graceful shutdown
	if *service == "server" {
		setup.StartServer(cacheClient, cc, store, pub, insights)
		return
	}

	// TODO: adicionar graceful shutdown
	if *service == "archived-notification" {
		setup.StartArchivedNotify(ctx, store, cacheClient)
		return
	}

	go setup.StartServer(cacheClient, cc, store, pub, insights)
	setup.StartWorker(clientPubsub, cc, store, pub, insights)

}
