package setup

import (
	"log"
	"net/http"

	"github.com/IsaacDSC/webhook/internal/backoffice"
	"github.com/IsaacDSC/webhook/internal/eventqueue"
	"github.com/IsaacDSC/webhook/internal/infra/middleware"
	"github.com/IsaacDSC/webhook/internal/interstore"
	cache2 "github.com/IsaacDSC/webhook/pkg/cache"
	"github.com/IsaacDSC/webhook/pkg/httpsvc"
	"github.com/IsaacDSC/webhook/pkg/publisher"
)

func StartServer(cache cache2.Cache, store interstore.Repository, pub publisher.Publisher) {
	mux := http.NewServeMux()

	routes := []httpsvc.HttpHandle{
		backoffice.CreateConsumer(cache, store),
		eventqueue.Publisher(pub),
	}

	for _, route := range routes {
		mux.HandleFunc(route.Path, route.Handler)
	}

	handler := middleware.LoggerMiddleware(mux)

	log.Println("Starting HTTP server on :8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		panic(err)
	}
}
