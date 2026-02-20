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

// waitForShutdown waits for SIGINT/SIGTERM and gracefully shuts down the provided servers.
func waitForShutdown(server *http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down servers...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if server != nil {
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Backoffice server shutdown error: %v", err)
		}
	}

	log.Println("Server shutdown complete")
}

// TODO: MODIFICAR NO MAKEFILE
// go run . --scope=all
// go run . --scope=backoffice
// go run . --scope=pubsub
// go run . --scope=task
func main() {
	conf := cfg.Get()
	ctx := context.Background()

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

		defer s.Close()

		servers = append(servers, s.Server())
	}

	if scopeOrAll(*scope, "task") {
		s := task.New(
			store, memStore, fetch, storeInsights,
		)

		s.Start(ctx, conf)

		defer s.Close()

		servers = append(servers, s.Server())
	}

	for _, server := range servers {
		waitForShutdown(server)
	}

}

func scopeOrAll(scope, expected string) bool {
	return scope == "all" || scope == expected
}
