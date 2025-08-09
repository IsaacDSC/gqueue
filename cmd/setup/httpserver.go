package setup

import (
	"log"
	"net/http"

	"github.com/IsaacDSC/gopherline/internal/backoffice"
	"github.com/IsaacDSC/gopherline/internal/eventqueue"
	"github.com/IsaacDSC/gopherline/internal/interstore"
	cache2 "github.com/IsaacDSC/gopherline/pkg/cache"
	"github.com/IsaacDSC/gopherline/pkg/httpsvc"
	"github.com/IsaacDSC/gopherline/pkg/publisher"
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

	handler := LoggerMiddleware(mux)

	log.Println("Starting HTTP server on :8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		panic(err)
	}
}
