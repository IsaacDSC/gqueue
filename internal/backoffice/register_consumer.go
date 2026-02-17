package backoffice

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/IsaacDSC/gqueue/internal/cfg"
	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/cachemanager"
	"github.com/IsaacDSC/gqueue/pkg/httpadapter"
	"github.com/google/uuid"
)

type EventDto struct {
	Name     string           `json:"name"`
	Type     domain.Type      `json:"type"`
	Triggers []domain.Trigger `json:"triggers"`
}

func (e *EventDto) ToDomain() domain.Event {
	env := cfg.Get()

	return domain.Event{
		ID:          uuid.New(),
		Name:        e.Name,
		ServiceName: env.InternalServiceName,
		Type:        e.Type,
		Triggers:    e.Triggers,
	}
}

func CreateConsumer(cc cachemanager.Cache, repo Repository) httpadapter.HttpHandle {
	return httpadapter.HttpHandle{
		Path: "POST /api/v1/event/consumer",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			var payload EventDto
			defer r.Body.Close()
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			event := payload.ToDomain()
			if err := event.Validate(); err != nil {
				http.Error(w, fmt.Sprintf("invalid event payload: %s", err.Error()), http.StatusBadRequest)
				return
			}

			ctx := r.Context()
			key := eventKey(cc, event.ServiceName, event.Name)
			defaultTTL := cc.GetDefaultTTL()

			if err := cc.Hydrate(ctx, key, &payload, defaultTTL, func(ctx context.Context) (any, error) {
				if err := repo.Save(ctx, event); err != nil {
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
