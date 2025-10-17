package backoffice

import (
	"context"
	"net/http"

	"github.com/IsaacDSC/gqueue/pkg/cachemanager"
	httpadapter "github.com/IsaacDSC/gqueue/pkg/httpadapter"
	"github.com/google/uuid"
)

func RemoveEvent(cc cachemanager.Cache, repo Repository) httpadapter.HttpHandle {
	return httpadapter.HttpHandle{
		Path: "DELETE /api/v1/event/{id}",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			eventID, err := uuid.Parse(r.PathValue("id"))
			if err != nil {
				http.Error(w, "Invalid event ID", http.StatusBadRequest)
				return
			}

			event, err := repo.GetEventByID(ctx, eventID)
			if err != nil {
				http.Error(w, "Event not found", http.StatusNotFound)
				return
			}

			key := eventKey(cc, event.ServiceName, event.Name)
			if err := cc.RemoveValue(ctx, key, func(ctx context.Context) error {
				return repo.DisabledEvent(ctx, eventID)
			}); err != nil {
				http.Error(w, "Failed to remove event", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		},
	}
}
