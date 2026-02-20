package backofficeapp

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/httpadapter"
)

type InsightsStore interface {
	GetAll(ctx context.Context) (domain.Metrics, error)
}

func GetInsightsHandle(store InsightsStore) httpadapter.HttpHandle {
	return httpadapter.HttpHandle{
		Path: "/api/v1/insights",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			metrics, err := store.GetAll(ctx)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(metrics.Insights())
		}),
	}
}
