package setup

import (
	"context"
	"time"

	"github.com/IsaacDSC/gqueue/internal/asynqstore"
	"github.com/IsaacDSC/gqueue/internal/asynqtask"
	"github.com/IsaacDSC/gqueue/internal/fetcher"
	"github.com/IsaacDSC/gqueue/internal/interstore"
	"github.com/redis/go-redis/v9"
)

func StartArchivedNotify(ctx context.Context, store interstore.Repository, cache *redis.Client) {
	cacheManager := asynqstore.NewCache(cache)
	fetch := fetcher.NewNotification()
	svc := asynqtask.NewTaskManager(store, cacheManager, fetch)

	for {
		svc.NotifyListeners(ctx)
		time.Sleep(30 * time.Second)
	}

}
