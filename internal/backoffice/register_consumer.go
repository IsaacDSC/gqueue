package backoffice

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/IsaacDSC/gqueue/internal/cfg"
	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/cachemanager"
	"github.com/IsaacDSC/gqueue/pkg/ctxlogger"
	"github.com/IsaacDSC/gqueue/pkg/httpadapter"
	"github.com/google/uuid"
)

type EventDto struct {
	Name      string            `json:"name"`
	Type      domain.Type       `json:"type"`
	Option    domain.Opt        `json:"option" bson:"option"`
	Consumers []domain.Consumer `json:"consumers"`
}

func (e *EventDto) ToDomain() domain.Event {
	env := cfg.Get()

	return domain.Event{
		ID:          uuid.New(),
		Name:        e.Name,
		ServiceName: env.InternalServiceName,
		Type:        e.Type,
		Option:      e.Option,
		Consumers:   e.Consumers,
	}
}

func PatchConsumer(cc cachemanager.Cache, repo Repository) httpadapter.HttpHandle {
	return httpadapter.HttpHandle{
		Path: "PATCH /api/v1/event/consumer",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			l := ctxlogger.GetLogger(ctx)

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

			key := eventKey(cc, event.ServiceName, event.Name)
			defaultTTL := cc.GetDefaultTTL()

			if err := cc.Hydrate(ctx, key, &payload, defaultTTL, func(ctx context.Context) (any, error) {
				if err := repo.Upsert(ctx, event); err != nil {
					return domain.Event{}, fmt.Errorf("failed to upsert internal event: %w", err)
				}
				return payload, nil
			}); err != nil {
				l.Error("failed to save consumer", "error", err)
				http.Error(w, "failed to save consumer", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(payload); err != nil {
				l.Error("failed to encode response", "error", err)
				http.Error(w, "failed to encode response", http.StatusInternalServerError)
			}
		},
	}
}
