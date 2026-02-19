package backoffice

import (
	"encoding/json"
	"net/http"

	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/httpadapter"
	"github.com/IsaacDSC/gqueue/pkg/queryparser"
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

func GetEvents(repo Repository) httpadapter.HttpHandle {
	return httpadapter.HttpHandle{
		Path: "GET /api/v1/events",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			var filter domain.FilterEvents

			defaults := map[string]any{
				"page":  uint(1),
				"limit": uint(500),
				"state": []string{"active"},
			}

			if err := queryparser.ParseQueryParamsWithDefaults(r.URL.Query(), &filter, defaults); err != nil {
				http.Error(w, "Invalid query parameters: "+err.Error(), http.StatusBadRequest)
				return
			}

			events, err := repo.GetInternalEvents(ctx, filter)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(events); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		},
	}
}
