package backoffice

import (
	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/cachemanager"
)

type Cache interface {
	Key(params ...string) cachemanager.Key
}

func eventKey(cache Cache, serviceName, eventName string) cachemanager.Key {
	return cache.Key(domain.CacheKeyEventPrefix, serviceName, eventName)
}
