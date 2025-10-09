package backoffice

import (
	"net/http"

	"github.com/IsaacDSC/gqueue/pkg/cachemanager"
	"github.com/IsaacDSC/gqueue/pkg/httpsvc"
	"github.com/google/uuid"
)

func RemoveEvent(cc cachemanager.Cache, repo Repository) httpsvc.HttpHandle {
	return httpsvc.HttpHandle{
		Path: "DELETE /api/v1/event/{id}",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			eventID, err := uuid.Parse(r.PathValue("id"))
			if err != nil {
				http.Error(w, "Invalid event ID", http.StatusBadRequest)
				return
			}

			if err := repo.DisabledEvent(ctx, eventID); err != nil {
				http.Error(w, "Failed to disable event", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		},
	}
}
