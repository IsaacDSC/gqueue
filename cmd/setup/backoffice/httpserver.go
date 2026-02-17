package backoffice

import (
	"context"
	"log"
	"net/http"

	"github.com/IsaacDSC/gqueue/cmd/setup/middleware"
	"github.com/IsaacDSC/gqueue/internal/backoffice"
	"github.com/IsaacDSC/gqueue/internal/cfg"
	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/internal/interstore"
	"github.com/IsaacDSC/gqueue/pkg/cachemanager"
	"github.com/IsaacDSC/gqueue/pkg/httpadapter"
	"github.com/IsaacDSC/gqueue/pkg/pubadapter"
	"github.com/redis/go-redis/v9"
)

type InsightsStore interface {
	GetAll(ctx context.Context) (output domain.Metrics, err error)
}

func Start(
	rdsclient *redis.Client,
	cache cachemanager.Cache,
	store interstore.Repository,
	pub pubadapter.Publisher,
	insightsStore InsightsStore,
) {
	mux := http.NewServeMux()

	routes := []httpadapter.HttpHandle{
		backoffice.GetHealthCheckHandler(),
		backoffice.CreateConsumer(cache, store),
		backoffice.GetEvent(cache, store),
		backoffice.GetEvents(cache, store),
		backoffice.GetPathEventHandle(cache, store),
		backoffice.GetRegisterTaskConsumerArchived(cache, store),
		backoffice.RemoveEvent(cache, store),
		backoffice.GetInsightsHandle(insightsStore),
	}

	for _, route := range routes {
		mux.HandleFunc(route.Path, route.Handler)
	}

	// config := cfg.Get()

	// authorization := auth.NewBasicAuth(map[string]string{
	// 	config.ProjectID: config.SecretKey,
	// })

	handler := middleware.CORSMiddleware(middleware.LoggerMiddleware(mux))
	// h := authorization.Middleware(handler.ServeHTTP)

	env := cfg.Get()
	port := env.BackofficePort

	log.Printf("Starting Backoffice server on :%d", port)

	if err := http.ListenAndServe(port.String(), handler); err != nil {
		panic(err)
	}
}
