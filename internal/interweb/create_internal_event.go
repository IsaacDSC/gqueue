package interweb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/IsaacDSC/webhook/internal/intersvc"
)

type CreateEventFunc func(ctx context.Context, input intersvc.CreateInternalEventDto) (output intersvc.InternalEvent, err error)

func GetCreateEventHandle(createEvent CreateEventFunc) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var body intersvc.CreateInternalEventDto

		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			fmt.Println("Error decoding request body:", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		internalEvent, err := createEvent(r.Context(), body)
		if err != nil {
			fmt.Println("Error creating internal event:", err)
			http.Error(w, "Failed to create internal event", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(internalEvent); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}
