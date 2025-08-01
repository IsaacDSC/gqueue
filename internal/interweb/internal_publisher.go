package interweb

import (
	"encoding/json"
	"net/http"

	"github.com/IsaacDSC/webhook/internal/consworker"
	"github.com/IsaacDSC/webhook/pkg/publisher"
)

type Payload struct {
	EventName string `json:"event_name"`
	Data      Data   `json:"data"`
}

type Data map[string]any

func (d Data) ToBytes() []byte {
	data, _ := json.Marshal(d)
	return data
}

func GetPublisherHandle(internpub publisher.Publisher) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload Payload

		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// b, _ := io.ReadAll(r.Body) // Read the body to avoid closing it prematurely
		// fmt.Println("Received request to publish event", string(b))

		if err := internpub.Publish(r.Context(), consworker.PublisherExternalEvent.String(), payload); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}
