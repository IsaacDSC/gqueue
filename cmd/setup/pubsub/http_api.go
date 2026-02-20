package pubsub

import (
	"context"
	"log"
	"net/http"

	"github.com/IsaacDSC/gqueue/cmd/setup/middleware"
	"github.com/IsaacDSC/gqueue/internal/app/backoffice"
	"github.com/IsaacDSC/gqueue/internal/app/pubsubapp"
	"github.com/IsaacDSC/gqueue/internal/cfg"
	"github.com/IsaacDSC/gqueue/pkg/httpadapter"
)

func (s *Service) startHttpServer(ctx context.Context, env cfg.Config) *http.Server {

	mux := http.NewServeMux()

	routes := []httpadapter.HttpHandle{
		backoffice.GetHealthCheckHandler(),
		pubsubapp.PublisherEvent(s.memStore, s.gcppublisher, s.insightsStore),
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

	port := env.PubsubApiPort

	server := &http.Server{
		Addr:    port.String(),
		Handler: handler,
	}

	log.Printf("[*] Starting Pubsub API server on :%d", port)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("API server error: %v", err)
		}
	}()

	return server
}
