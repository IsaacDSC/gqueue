package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/IsaacDSC/gqueue/cmd/setup/backoffice"
	"github.com/IsaacDSC/gqueue/cmd/setup/pubsub"
	"github.com/IsaacDSC/gqueue/cmd/setup/task"
	"github.com/IsaacDSC/gqueue/internal/cfg"
	"github.com/IsaacDSC/gqueue/internal/fetcher"
	"github.com/IsaacDSC/gqueue/internal/interstore"
	"github.com/IsaacDSC/gqueue/internal/storests"
	"github.com/redis/go-redis/v9"
)

// Centralized shutdown: creates a context that is cancelled on SIGINT/SIGTERM and waits for all servers to shutdown.
func waitForShutdown(ctx context.Context, servers []*http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		log.Println("Shutting down servers...")
	case <-ctx.Done():
		log.Println("Context cancelled, shutting down servers...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	for _, server := range servers {
		if server != nil {
			if err := server.Shutdown(shutdownCtx); err != nil {
				log.Printf("Server shutdown error: %v", err)
			}
		}
	}

	log.Println("All servers shutdown complete")
}

// go run . --scope=all
// go run . --scope=backoffice
// go run . --scope=pubsub
func main() {
	conf := cfg.Get()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scope := flag.String("scope", "all", "service to run")
	flag.Parse()

	redisClient := redis.NewClient(&redis.Options{Addr: conf.Cache.CacheAddr})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		panic(err)
	}

	storeInsights := storests.NewStore(redisClient)

	store, err := interstore.NewPostgresStoreFromDSN(conf.ConfigDatabase.DbConn)
	if err != nil {
		panic(err)
	}

	var servers []*http.Server
	var closers []func()

	if scopeOrAll(*scope, "backoffice") {
		backofficeServer := backoffice.Start(
			redisClient,
			store,
			storeInsights,
		)
		servers = append(servers, backofficeServer)
	}

	var memStore *interstore.MemStore
	var fetch *fetcher.Notification
	// task and pubsub share some dependencies, so we initialize them here and pass to both services
	if *scope == "pubsub" || *scope == "task" || *scope == "all" {
		memStore = interstore.NewMemStore(store)
		fetch = fetcher.NewNotification()
	}

	if scopeOrAll(*scope, "pubsub") {
		s := pubsub.New(
			store, memStore, fetch, storeInsights,
		)
		s.Start(ctx, conf)
		closers = append(closers, s.Close)
		servers = append(servers, s.Server())
	}

	if scopeOrAll(*scope, "task") {
		s := task.New(
			store, memStore, fetch, storeInsights,
		)
		s.Start(ctx, conf)
		closers = append(closers, s.Close)
		servers = append(servers, s.Server())
	}

	waitForShutdown(ctx, servers)

	// call all closers after shutdown
	for _, closeFn := range closers {
		closeFn()
	}
}

func scopeOrAll(scope, expected string) bool {
	return scope == "all" || scope == expected
}
