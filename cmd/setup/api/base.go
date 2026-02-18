package api

import (
	"context"
	"log"
	"net/http"

	"cloud.google.com/go/pubsub"
	"github.com/IsaacDSC/gqueue/cmd/setup/middleware"
	"github.com/IsaacDSC/gqueue/internal/backoffice"
	"github.com/IsaacDSC/gqueue/internal/cfg"
	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/internal/fetcher"
	"github.com/IsaacDSC/gqueue/internal/storests"
	"github.com/IsaacDSC/gqueue/internal/wtrhandler"
	"github.com/IsaacDSC/gqueue/pkg/httpadapter"
	"github.com/IsaacDSC/gqueue/pkg/pubadapter"
	"github.com/hibiken/asynq"
)

type PersistentRepository interface {
	GetAllEvents(ctx context.Context) ([]domain.Event, error)
	GetAllSchedulers(ctx context.Context, state string) ([]domain.Event, error)
}

func Start(
	ctx context.Context,
	store PersistentRepository,
	asynqClient *asynq.Client,
	gcppubsubClient *pubsub.Client,
	insightsStore *storests.Store,
) *http.Server {
	fetch := fetcher.NewNotification()

	memStore := loadInMemStore(store)

	classificationResult := pubadapter.ClassificationPublisher(
		pubadapter.NewPubSubGoogle(gcppubsubClient),
		pubadapter.NewPublisher(asynqClient),
	)

	adaptpub := pubadapter.NewStrategy(&classificationResult)

	if gcppubsubClient != nil {
		go startUsingGooglePubSub(
			memStore,
			gcppubsubClient,
			adaptpub,
			fetch, insightsStore,
		)
	}

	if asynqClient != nil {
		go startUsingAsynq(memStore, adaptpub, fetch, insightsStore)
	}

	StartTaskSyncMemStore(ctx, store, memStore)

	mux := http.NewServeMux()

	routes := []httpadapter.HttpHandle{
		backoffice.GetHealthCheckHandler(),
		wtrhandler.PublisherEvent(memStore, adaptpub, insightsStore),
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
	port := env.ApiPort

	server := &http.Server{
		Addr:    port.String(),
		Handler: handler,
	}

	log.Printf("Starting API server on :%d", port)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("API server error: %v", err)
		}
	}()

	return server
}
