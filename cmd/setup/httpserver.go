package setup

import (
	"log"
	"net/http"

	"github.com/IsaacDSC/gqueue/internal/backoffice"
	"github.com/IsaacDSC/gqueue/internal/eventqueue"
	"github.com/IsaacDSC/gqueue/internal/interstore"
	cache2 "github.com/IsaacDSC/gqueue/pkg/cache"
	"github.com/IsaacDSC/gqueue/pkg/httpsvc"
	"github.com/IsaacDSC/gqueue/pkg/publisher"
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
