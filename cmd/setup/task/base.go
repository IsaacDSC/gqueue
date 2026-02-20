package task

import (
	"context"
	"net/http"
	"time"

	"github.com/IsaacDSC/gqueue/internal/cfg"
	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/internal/fetcher"
	"github.com/IsaacDSC/gqueue/internal/interstore"
	"github.com/IsaacDSC/gqueue/internal/storests"
	"github.com/IsaacDSC/gqueue/pkg/ctxlogger"
	"github.com/IsaacDSC/gqueue/pkg/pubadapter"
	"github.com/hibiken/asynq"
)

type PersistentRepository interface {
	GetAllEvents(ctx context.Context) ([]domain.Event, error)
	GetAllSchedulers(ctx context.Context, state string) ([]domain.Event, error)
}

type Service struct {
	asynqClient    *asynq.Client
	asynqServer    *asynq.Server
	asynqPublisher pubadapter.GenericPublisher
	server         *http.Server
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
	s.asynqClient = asynq.NewClient(asynq.RedisClientOpt{Addr: env.Cache.CacheAddr})

	asynqCfg := asynq.Config{
		Concurrency: env.AsynqConfig.Concurrency,
	}

	s.asynqServer = asynq.NewServer(
		asynq.RedisClientOpt{Addr: env.Cache.CacheAddr},
		asynqCfg,
	)

	go s.consumer(ctx, env, asynqCfg)

	s.asynqPublisher = pubadapter.NewPublisher(s.asynqClient)

	// load mem store with events from persistent store
	s.memStore.LoadInMemStore(ctx)

	// task refresh mem store
	go func() {
		l := ctxlogger.GetLogger(ctx)
		trigger := time.NewTicker(time.Minute)
		for {
			select {
			case <-trigger.C:
				if err := s.memStore.LoadInMemStore(ctx); err != nil {
					l.Error("Error refreshing mem store with events from persistent store", "error", err)
					continue
				}

				l.Info("Executed periodic refresh of mem store with events from persistent store", "scope", "task")
			case <-ctx.Done():
				trigger.Stop()
				return
			}
		}
	}()

	s.server = s.startHttpServer(ctx, env)
}

func (s *Service) Close() { _ = s.asynqClient.Close() }

func (s *Service) Server() *http.Server { return s.server }
