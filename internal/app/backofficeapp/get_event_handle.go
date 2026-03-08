package backofficeapp

import (
	"encoding/json"
	"net/http"

	"github.com/IsaacDSC/gqueue/pkg/httpadapter"
)

func GetEvent(repo Repository) httpadapter.HttpHandle {
	return httpadapter.HttpHandle{
		Path: "GET /api/v1/events/{event_name}",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			eventName := r.PathValue("event_name")

			event, err := repo.GetInternalEvent(ctx, eventName)
			if err != nil {
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
