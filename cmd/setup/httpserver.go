package setup

import (
	"context"
	"log"
	"net/http"

	"github.com/IsaacDSC/gqueue/internal/backoffice"
	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/internal/interstore"
	"github.com/IsaacDSC/gqueue/internal/wtrhandler"
	cache2 "github.com/IsaacDSC/gqueue/pkg/cachemanager"
	"github.com/IsaacDSC/gqueue/pkg/httpsvc"
	"github.com/IsaacDSC/gqueue/pkg/publisher"
	"github.com/redis/go-redis/v9"
)

type InsightsStore interface {
	GetAll(ctx context.Context) (output domain.Metrics, err error)
}

func StartServer(
	rdsclient *redis.Client,
	cache cache2.Cache,
	store interstore.Repository,
	pub publisher.Publisher,
	insightsStore InsightsStore,
) {
	mux := http.NewServeMux()

	routes := []httpsvc.HttpHandle{
		backoffice.GetHealthCheckHandler(),
		backoffice.CreateConsumer(cache, store),
		backoffice.GetEvent(cache, store),
		backoffice.GetEvents(cache, store),
		backoffice.GetRegisterTaskConsumerArchived(cache, store),
		backoffice.RemoveEvent(cache, store),
		backoffice.GetInsightsHandle(insightsStore),
		wtrhandler.Publisher(pub),
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
