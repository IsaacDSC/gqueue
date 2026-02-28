package backoffice

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/IsaacDSC/gqueue/cmd/setup/middleware"
	"github.com/IsaacDSC/gqueue/internal/app/backofficeapp"
	"github.com/IsaacDSC/gqueue/internal/app/health"
	"github.com/IsaacDSC/gqueue/internal/cfg"
	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/internal/interstore"
	"github.com/IsaacDSC/gqueue/pkg/httpadapter"
	"github.com/redis/go-redis/v9"
)

type InsightsStore interface {
	GetAll(ctx context.Context) (output domain.Metrics, err error)
}

func Start(
	rdsclient *redis.Client,
	store interstore.Repository,
	insightsStore InsightsStore,
) *http.Server {
	mux := http.NewServeMux()

	routes := []httpadapter.HttpHandle{
		health.GetHealthCheckHandler(),
		backofficeapp.PatchConsumer(store),
		backofficeapp.GetEvent(store),
		backofficeapp.GetEvents(store),
		backofficeapp.GetRegisterTaskConsumerArchived(store),
		backofficeapp.RemoveEvent(store),
		backofficeapp.GetInsightsHandle(insightsStore),
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
	port := env.BackofficeApiPort

	server := &http.Server{
		Addr:         port.String(),
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("[*] Starting Backoffice server on :%d", port)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Backoffice server error: %v", err)
		}
	}()

	return server
}
