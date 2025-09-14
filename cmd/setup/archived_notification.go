package setup

import (
	"context"
	"time"

	"github.com/IsaacDSC/gqueue/internal/fetcher"
	"github.com/IsaacDSC/gqueue/internal/task"
	"github.com/IsaacDSC/gqueue/internal/taskstore"
	"github.com/redis/go-redis/v9"
)

func StartArchivedNotify(ctx context.Context, cache *redis.Client) {
	store := taskstore.NewCache(cache)
	fetch := fetcher.NewNotification()
	svc := task.NewTaskManager(store, fetch)

	for {
		svc.NotifyListeners(ctx)
		time.Sleep(30 * time.Second)
	}

}
