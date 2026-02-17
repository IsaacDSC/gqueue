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
)

type PersistentRepository interface {
	GetAllEvents(ctx context.Context) ([]domain.Event, error)
	GetAllSchedulers(ctx context.Context, state string) ([]domain.Event, error)
}

func Start(
	ctx context.Context,
	store PersistentRepository,
	clientPubsub *pubsub.Client,
	redisAsync pubadapter.GenericPublisher,
	insightsStore *storests.Store,
) {
	fetch := fetcher.NewNotification()

	memStore := loadInMemStore(store)

	if clientPubsub != nil {
		go startUsingGooglePubSub(clientPubsub, memStore, redisAsync, fetch, insightsStore)
	}

	if redisAsync != nil {
		go startUsingAsynq(memStore, redisAsync, fetch, insightsStore)
	}

	StartTaskSyncMemStore(ctx, store, memStore)

	mux := http.NewServeMux()

	routes := []httpadapter.HttpHandle{
		backoffice.GetHealthCheckHandler(),
		wtrhandler.Publisher(redisAsync), //TODO: inject abstract gcp.pubsub or asynq
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

	log.Printf("Starting API server on :%d", port)

	if err := http.ListenAndServe(port.String(), handler); err != nil {
		panic(err)
	}
}
