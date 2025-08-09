package eventqueue

import (
	"encoding/json"
	"net/http"

	"github.com/IsaacDSC/gqueue/pkg/httpsvc"
	"github.com/IsaacDSC/gqueue/pkg/publisher"
)

type ExternalPayload struct {
	EventName string            `json:"event_name"`
	Data      map[string]any    `json:"data"`
	Headers   map[string]string `json:"headers"`
	Trigger   Trigger           `json:"trigger"`
}

func Publisher(pub publisher.Publisher) httpsvc.HttpHandle {
	return httpsvc.HttpHandle{
		Path: "POST /event/publisher",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			var payload InternalPayload

			defer r.Body.Close()
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			if err := payload.Validate(); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			conf := payload.GetOpts()
			if err := pub.Publish(r.Context(), "event-queue.internal", payload, conf...); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusAccepted)
		},
	}
}
