package pubsub

import (
	"context"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/pubsub"
	vkit "cloud.google.com/go/pubsub/apiv1"
	"github.com/IsaacDSC/gqueue/cmd/setup/memstore"
	"github.com/IsaacDSC/gqueue/internal/cfg"
	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/internal/fetcher"
	"github.com/IsaacDSC/gqueue/internal/interstore"
	"github.com/IsaacDSC/gqueue/internal/storests"
	"github.com/IsaacDSC/gqueue/pkg/pubadapter"
	"github.com/googleapis/gax-go/v2"
	"google.golang.org/grpc/codes"
)

type PersistentRepository interface {
	GetAllEvents(ctx context.Context) ([]domain.Event, error)
	GetAllSchedulers(ctx context.Context, state string) ([]domain.Event, error)
}

type Service struct {
	pubsubClient *pubsub.Client
	gcppublisher pubadapter.GenericPublisher
	server       *http.Server
	// injectable dependencies
	persistentStore PersistentRepository
	memStore        *interstore.MemStore
	fetch           *fetcher.Notification
	insightsStore   *storests.Store
}

func New(
	ps PersistentRepository,
	ms *interstore.MemStore,
	fetch *fetcher.Notification,
	insightsStore *storests.Store,
) *Service {
	return &Service{
		persistentStore: ps,
		memStore:        ms,
		fetch:           fetch,
		insightsStore:   insightsStore,
	}
}

func (s *Service) Start(ctx context.Context, env cfg.Config) {
	config := &pubsub.ClientConfig{
		PublisherCallOptions: &vkit.PublisherCallOptions{
			Publish: []gax.CallOption{
				gax.WithRetry(func() gax.Retryer {
					return gax.OnCodes([]codes.Code{
						codes.Aborted,
						codes.Canceled,
						codes.Internal,
						codes.ResourceExhausted,
						codes.Unknown,
						codes.Unavailable,
						codes.DeadlineExceeded,
					}, gax.Backoff{
						Initial:    250 * time.Millisecond, // default 100 milliseconds
						Max:        5 * time.Second,        // default 60 seconds
						Multiplier: 1.45,                   // default 1.3
					})
				}),
			},
		},
	}

	clientPubsub, err := pubsub.NewClientWithConfig(ctx, domain.ProjectID, config)
	if err != nil {
		log.Fatalf("Erro ao criar cliente: %v", err)
	}

	s.pubsubClient = clientPubsub

	// setup publisher
	s.gcppublisher = pubadapter.NewPubSubGoogle(s.pubsubClient)

	// setup consumer depends on publisher
	go s.consumer(ctx, env)

	// load mem store with events from persistent store
	s.memStore.LoadInMemStore(ctx)

	// task refresh mem store
	go memstore.SyncMemStore(ctx, s.memStore)

	s.server = s.startHttpServer(ctx, env)
}

func (s *Service) Close() { _ = s.pubsubClient.Close() }

func (s *Service) Server() *http.Server { return s.server }
