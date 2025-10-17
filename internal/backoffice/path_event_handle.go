package backoffice

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/cachemanager"
	"github.com/IsaacDSC/gqueue/pkg/httpadapter"
	"github.com/google/uuid"
)

func GetPathEventHandle(cc cachemanager.Cache, repo Repository) httpadapter.HttpHandle {
	return httpadapter.HttpHandle{
		Path: "PATCH /api/v1/event/{id}",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			id, err := uuid.Parse(r.PathValue("id"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			defer r.Body.Close()
			var payload domain.Event
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			payload.ID = id

			key := eventKey(cc, payload.ServiceName, payload.Name)
			defaultTTL := cc.GetDefaultTTL()

			if err := cc.Hydrate(ctx, key, &payload, defaultTTL, func(ctx context.Context) (any, error) {
				return payload, repo.UpdateEvent(ctx, payload)
			}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err := json.NewEncoder(w).Encode(payload); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		},
	}
}
