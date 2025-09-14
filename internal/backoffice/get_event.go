package backoffice

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/cachemanager"
	"github.com/IsaacDSC/gqueue/pkg/httpsvc"
)

func GetEvents(cc cachemanager.Cache, repo Repository) httpsvc.HttpHandle {
	return httpsvc.HttpHandle{
		Path: "GET /events/{service_name}",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			serviceName := r.PathValue("service_name")

			eventName := r.URL.Query().Get("event_name")
			key := eventKey(cc, serviceName, eventName)
			defaultTTL := cc.GetDefaultTTL()

			var event []domain.Event
			if err := cc.Once(ctx, key, &event, defaultTTL, func(ctx context.Context) (any, error) {
				return repo.GetInternalEvent(ctx, eventName, serviceName)
			}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(event); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		},
	}
}
