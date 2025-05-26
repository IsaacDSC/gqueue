package handler

import (
	"encoding/json"
	"fmt"
	"github.com/IsaacDSC/webhook/internal/structs"
	"net/http"
)

func (h Handler) CreateInternalEvent(w http.ResponseWriter, r *http.Request) {
	var body structs.CreateInternalEventDto

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		fmt.Println("Error decoding request body:", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	internalEvent, err := h.service.CreateInternalEvent(r.Context(), body)
	if err != nil {
		http.Error(w, "Failed to create internal event", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(internalEvent); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
