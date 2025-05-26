package handler

import (
	"encoding/json"
	"github.com/IsaacDSC/webhook/internal/structs"
	"net/http"
)

func (h Handler) PublisherExternalEvent(w http.ResponseWriter, r *http.Request) {
	var body structs.PublisherExternalEventDto
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.service.PublisherExternalEvent(r.Context(), body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
