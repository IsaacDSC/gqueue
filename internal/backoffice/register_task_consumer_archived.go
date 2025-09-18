package backoffice

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/cachemanager"
	"github.com/IsaacDSC/gqueue/pkg/httpsvc"
)

func GetRegisterTaskConsumerArchived(cc cachemanager.Cache, repo Repository) httpsvc.HttpHandle {
	return httpsvc.HttpHandle{
		Path: "POST /events/schedule/archived",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			defer r.Body.Close()

			var payload domain.Event
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			payload.TypeEvent = domain.TypeEventSchedule
			payload.State = "archived"

			key := cc.Key(payload.TypeEvent.String(), payload.State, payload.ServiceName, payload.Name)

			if err := cc.Hydrate(ctx, key, &payload, cc.GetDefaultTTL(), func(ctx context.Context) (any, error) {
				if err := repo.Save(ctx, payload); err != nil {
					return domain.Event{}, fmt.Errorf("failed to create internal event: %w", err)
				}
				return payload, nil
			}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			consumersKey := cc.Key("consumers", payload.TypeEvent.String(), payload.State)
			cc.IncrementValue(ctx, consumersKey, &payload)

			w.WriteHeader(http.StatusCreated)
		},
	}
}
