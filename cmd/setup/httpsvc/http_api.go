package httpsvc

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/IsaacDSC/gqueue/cmd/setup/middleware"
	"github.com/IsaacDSC/gqueue/internal/app/health"
	"github.com/IsaacDSC/gqueue/internal/cfg"
	"github.com/IsaacDSC/gqueue/pkg/httpadapter"
	"github.com/IsaacDSC/gqueue/pkg/telemetry"
)

func StartHttpServer(ctx context.Context, env cfg.Config, routes []httpadapter.HttpHandle, port string, serviceName string) *http.Server {

	mux := http.NewServeMux()

	// Rota de métricas para Prometheus.
	mux.Handle("/metrics", telemetry.Handler())

	routes = append(routes, health.GetHealthCheckHandler())

	for _, route := range routes {
		mux.HandleFunc(route.Path, route.Handler)
	}

	// config := cfg.Get()

	// authorization := auth.NewBasicAuth(map[string]string{
	// 	config.ProjectID: config.SecretKey,
	// })

	handler := middleware.CORSMiddleware(
		middleware.MetricsMiddleware(serviceName, middleware.LoggerMiddleware(mux)),
	)
	// h := authorization.Middleware(handler.ServeHTTP)

	server := &http.Server{
		Addr:         port,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 200 * time.Millisecond,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("[*] Starting API server on %s", port)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("API server error: %v", err)
		}
	}()

	return server
}
