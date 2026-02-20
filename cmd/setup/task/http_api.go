package task

import (
	"context"
	"log"
	"net/http"

	"github.com/IsaacDSC/gqueue/cmd/setup/middleware"
	"github.com/IsaacDSC/gqueue/internal/app/backoffice"
	"github.com/IsaacDSC/gqueue/internal/app/taskapp"
	"github.com/IsaacDSC/gqueue/internal/cfg"
	"github.com/IsaacDSC/gqueue/pkg/httpadapter"
)

func (s *Service) startHttpServer(ctx context.Context, env cfg.Config) *http.Server {

	mux := http.NewServeMux()

	routes := []httpadapter.HttpHandle{
		backoffice.GetHealthCheckHandler(),
		taskapp.PublisherEvent(s.memStore, s.asynqPublisher, s.insightsStore),
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

	port := env.TaskApiPort

	server := &http.Server{
		Addr:    port.String(),
		Handler: handler,
	}

	log.Printf("[*] Starting Task API server on :%d", port)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("API server error: %v", err)
		}
	}()

	return server
}
