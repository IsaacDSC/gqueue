package backoffice

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/IsaacDSC/gopherline/internal/domain"
	"github.com/IsaacDSC/gopherline/pkg/cache"
	"github.com/IsaacDSC/gopherline/pkg/httpsvc"
)

type Repository interface {
	Save(ctx context.Context, event domain.Event) error
}

func CreateConsumer(cc cache.Cache, repo Repository) httpsvc.HttpHandle {
	return httpsvc.HttpHandle{
		Path: "POST /event/consumer",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			var payload domain.Event
			defer r.Body.Close()
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			if err := payload.Validate(domain.ValidateTypeExternal); err != nil {
				http.Error(w, fmt.Sprintf("invalid event payload: %s", err.Error()), http.StatusBadRequest)
				return
			}

			ctx := r.Context()
			key := cc.Key(domain.CacheKeyEventPrefix, payload.Name)
			defaultTTL := cc.GetDefaultTTL()

			if err := cc.Hydrate(ctx, key, &payload, defaultTTL, func(ctx context.Context) (any, error) {
				if err := repo.Save(ctx, payload); err != nil {
					return domain.Event{}, fmt.Errorf("failed to create internal event: %w", err)
				}
				return payload, nil
			}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusCreated)
		},
	}
}
