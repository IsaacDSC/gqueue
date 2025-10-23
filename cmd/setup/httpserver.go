package setup

import (
	"context"
	"log"
	"net/http"

	"github.com/IsaacDSC/gqueue/internal/backoffice"
	"github.com/IsaacDSC/gqueue/internal/cfg"
	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/internal/interstore"
	"github.com/IsaacDSC/gqueue/internal/wtrhandler"
	"github.com/IsaacDSC/gqueue/pkg/auth"
	"github.com/IsaacDSC/gqueue/pkg/cachemanager"
	"github.com/IsaacDSC/gqueue/pkg/httpadapter"
	"github.com/IsaacDSC/gqueue/pkg/pubadapter"
	"github.com/redis/go-redis/v9"
)

type InsightsStore interface {
	GetAll(ctx context.Context) (output domain.Metrics, err error)
}

func StartServer(
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
		wtrhandler.Publisher(pub),
	}

	for _, route := range routes {
		mux.HandleFunc(route.Path, route.Handler)
	}

	config := cfg.Get()

	authorization := auth.NewBasicAuth(map[string]string{
		config.ProjectID: config.SecretKey,
	})

	handler := CORSMiddleware(LoggerMiddleware(mux))
	h := authorization.Middleware(handler.ServeHTTP)

	log.Println("Starting HTTP server on :8080")
	if err := http.ListenAndServe(":8080", h); err != nil {
		panic(err)
	}
}
