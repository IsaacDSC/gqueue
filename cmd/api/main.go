package main

import (
	"context"
	"flag"
	"github.com/IsaacDSC/webhook/cmd/setup"
	"github.com/IsaacDSC/webhook/internal/infra/cfg"
	"github.com/IsaacDSC/webhook/internal/infra/repository"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// go run . --service=worker
// go run . --service=webhook
// go run . --service=all
func main() {
	cfg := cfg.Get()

	client, err := mongo.Connect(options.Client().ApplyURI(cfg.ConfigDatabase.DbConn))
	if err != nil {
		panic(err)
	}

	defer func() {
		if err = client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()

	repository := repository.NewRepository(client)

	service := flag.String("service", "all", "service to run")
	flag.Parse()

	if *service == "worker" {
		setup.StartWorker(repository)
	}

	if *service == "webhook" {
		setup.StartServer(repository)
	}

	go setup.StartServer(repository)
	setup.StartWorker(repository)

}
