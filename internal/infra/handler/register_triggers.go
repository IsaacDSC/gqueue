package handler

import (
	"encoding/json"
	"github.com/IsaacDSC/webhook/internal/structs"
	"net/http"
)

func (h Handler) RegisterTrigger(w http.ResponseWriter, r *http.Request) {
	var dto structs.RegisterTriggersDto
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	internalEvent, err := h.service.RegisterTrigger(r.Context(), dto)
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
