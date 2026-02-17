package api

import (
	"context"
	"log"
	"time"

	"github.com/IsaacDSC/gqueue/internal/interstore"
	"github.com/IsaacDSC/gqueue/pkg/ctxlogger"
)

func StartTaskSyncMemStore(ctx context.Context, store PersistentRepository, memStore *interstore.MemStore) {
	l := ctxlogger.GetLogger(ctx)

	// refresh events in memory store every minute
	go func() {
		l.Info("Starting task to sync mem store with persistent store")
		trigger := time.NewTicker(time.Minute)
		for {
			select {
			case <-trigger.C:
				events, err := store.GetAllEvents(ctx)
				if err != nil {
					l.Warn("error fetching events from persistent store", "error", err)
					continue
				}

				memStore.Refresh(ctx, events)
			case <-ctx.Done():
				trigger.Stop()
				return
			}
		}
	}()

	// refresh schedulers in memory store every 5 minutes
	go func() {
		log.Println("Starting task to sync mem store with persistent store for schedulers")
		trigger := time.NewTicker(5 * time.Minute)
		for {
			select {
			case <-trigger.C:
				events, err := store.GetAllSchedulers(ctx, "archived")
				if err != nil {
					continue
				}

				memStore.RefreshRetryTopics(ctx, events)
			case <-ctx.Done():
				trigger.Stop()
				return
			}
		}
	}()

}
