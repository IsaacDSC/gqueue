package backofficeapp

import (
	"encoding/json"
	"net/http"

	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/ctxlogger"
	"github.com/IsaacDSC/gqueue/pkg/httpadapter"
)

func GetRegisterTaskConsumerArchived(repo Repository) httpadapter.HttpHandle {
	return httpadapter.HttpHandle{
		Path: "POST /events/schedule/archived",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			l := ctxlogger.GetLogger(ctx)
			defer r.Body.Close()

			var payload domain.Event
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			payload.State = "archived"

			if err := repo.Upsert(ctx, payload); err != nil {
				l.Error("failed to upsert internal event", "error", err)
				http.Error(w, "failed to upsert internal event", http.StatusInternalServerError)
			}

			w.WriteHeader(http.StatusCreated)
		},
	}
}
