package memstore

import (
	"context"
	"time"

	"github.com/IsaacDSC/gqueue/internal/interstore"
	"github.com/IsaacDSC/gqueue/pkg/ctxlogger"
)

func SyncMemStore(ctx context.Context, memStore *interstore.MemStore) {
	l := ctxlogger.GetLogger(ctx)
	trigger := time.NewTicker(time.Minute)
	for {
		select {
		case <-trigger.C:
			if err := memStore.LoadInMemStore(ctx); err != nil {
				l.Error("Error refreshing mem store with events from persistent store", "error", err)
				continue
			}

			l.Debug("Executed periodic refresh of mem store with events from persistent store", "scope", "pubsub")
		case <-ctx.Done():
			trigger.Stop()
			return
		}
	}
}
