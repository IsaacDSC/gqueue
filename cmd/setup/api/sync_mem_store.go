package api

import (
	"context"
	"fmt"
	"time"

	"github.com/IsaacDSC/gqueue/internal/interstore"
)

func StartTaskSyncMemStore(ctx context.Context, store PersistentRepository, memStore *interstore.MemStore) {
	// refresh events in memory store every minute
	go func() {
		fmt.Println("Starting task to sync mem store with persistent store")
		trigger := time.NewTicker(time.Minute)
		for {
			select {
			case <-trigger.C:
				events, err := store.GetAllEvents(ctx)
				if err != nil {
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
		fmt.Println("Starting task to sync mem store with persistent store for schedulers")
		trigger := time.NewTicker(5 * time.Minute)
		for {
			select {
			case <-trigger.C:
				events, err := store.GetAllSchedulers(ctx, "archived")
				if err != nil {
					continue
				}

				memStore.Refresh(ctx, events)
			case <-ctx.Done():
				trigger.Stop()
				return
			}
		}
	}()

}
