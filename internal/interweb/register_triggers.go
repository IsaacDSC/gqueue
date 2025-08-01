package interweb

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/IsaacDSC/webhook/internal/intersvc"
)

type RegisterTriggerFunc func(ctx context.Context, input intersvc.RegisterTriggersDto) (output intersvc.InternalEvent, err error)

func GetRegisterHandle(registerTrigger RegisterTriggerFunc) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var dto intersvc.RegisterTriggersDto
		if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		defer r.Body.Close()

		internalEvent, err := registerTrigger(r.Context(), dto)
		if err != nil {
			http.Error(w, "Failed to register trigger", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(internalEvent); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}
