package backoffice

import (
	"net/http"

	"github.com/IsaacDSC/gqueue/pkg/ctxlogger"
	httpadapter "github.com/IsaacDSC/gqueue/pkg/httpadapter"
	"github.com/google/uuid"
)

func RemoveEvent(repo Repository) httpadapter.HttpHandle {
	return httpadapter.HttpHandle{
		Path: "DELETE /api/v1/event/{id}",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			l := ctxlogger.GetLogger(ctx)
			eventID, err := uuid.Parse(r.PathValue("id"))
			if err != nil {
				http.Error(w, "Invalid event ID", http.StatusBadRequest)
				return
			}

			if err := repo.DisabledEvent(ctx, eventID); err != nil {
				l.Error("failed to disable event", "error", err)
				http.Error(w, "Failed to remove event", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		},
	}
}
