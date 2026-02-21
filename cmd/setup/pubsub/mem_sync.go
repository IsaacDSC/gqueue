package pubsub

import (
	"context"
	"time"

	"github.com/IsaacDSC/gqueue/pkg/ctxlogger"
)

func (s *Service) syncMemStore(ctx context.Context) {
	l := ctxlogger.GetLogger(ctx)
	trigger := time.NewTicker(time.Minute)
	for {
		select {
		case <-trigger.C:
			if err := s.memStore.LoadInMemStore(ctx); err != nil {
				l.Error("Error refreshing mem store with events from persistent store", "error", err)
				continue
			}

			l.Info("Executed periodic refresh of mem store with events from persistent store", "scope", "pubsub")
		case <-ctx.Done():
			trigger.Stop()
			return
		}
	}
}
