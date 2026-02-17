package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
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
	"github.com/IsaacDSC/gqueue/cmd/setup/api"
	"github.com/IsaacDSC/gqueue/cmd/setup/backoffice"
	"github.com/IsaacDSC/gqueue/internal/interstore"
	"github.com/IsaacDSC/gqueue/pkg/cachemanager"
	"github.com/IsaacDSC/gqueue/pkg/pubadapter"
	"github.com/redis/go-redis/v9"
)

const appName = "gqueue"

// TODO: rename to --scope=...
// go run . --service=all
// go run . --service=backoffice
// go run . --service=api
// go run . --service=archived-notification
func main() {
	conf := cfg.Get()
	ctx := context.Background()

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: conf.Cache.CacheAddr})
	defer asynqClient.Close()

	var highPerformancePublisher pubadapter.GenericPublisher
	var highPerformanceAsyncClient *pubsub.Client
	if conf.WQ == cfg.WQGooglePubSub {
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

		highPerformancePublisher = pubadapter.NewPubSubGoogle(clientPubsub)
		highPerformanceAsyncClient = clientPubsub

		defer clientPubsub.Close()
	}

	cacheClient := redis.NewClient(&redis.Options{Addr: conf.Cache.CacheAddr})
	if err := cacheClient.Ping(ctx).Err(); err != nil {
		panic(err)
	}

	storeInsights := storests.NewStore(cacheClient)

	store, err := interstore.NewPostgresStoreFromDSN(conf.ConfigDatabase.DbConn)
	if err != nil {
		panic(err)
	}

	cc := cachemanager.NewStrategy(appName, cacheClient)

	mediumPerformancePublisher := pubadapter.NewPublisher(asynqClient)

	pub := pubadapter.NewPub(highPerformancePublisher, mediumPerformancePublisher, conf.WQ)

	service := flag.String("service", "all", "service to run")
	flag.Parse()

	// TODO: adicionar graceful shutdown
	if *service == "api" {
		api.Start(
			ctx,
			store,
			highPerformanceAsyncClient,
			mediumPerformancePublisher,
			storeInsights,
		)
		return
	}

	// TODO: adicionar graceful shutdown
	if *service == "backoffice" {
		backoffice.Start(
			cacheClient,
			cc,
			store,
			pub,
			storeInsights,
		)
		return
	}

	// TODO: adicionar graceful shutdown
	if *service == "archived-notification" {
		setup.StartArchivedNotify(ctx, store, cacheClient)
		return
	}

	go backoffice.Start(
		cacheClient,
		cc,
		store,
		pub,
		storeInsights,
	)

	go api.Start(
		ctx,
		store,
		highPerformanceAsyncClient,
		mediumPerformancePublisher,
		storeInsights,
	)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down servers...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	<-shutdownCtx.Done()
	log.Println("Server shutdown complete")
}
