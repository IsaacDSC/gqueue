package interweb

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/IsaacDSC/webhook/internal/intersvc"
)

type BulkEventDto struct {
	Name        string    `json:"name"`
	ServiceName string    `json:"serviceName"`
	RepoURL     string    `json:"repoUrl"`
	TeamOwner   string    `json:"teamOwner"`
	Triggers    []Trigger `json:"triggers"`
}

func (dto BulkEventDto) ToInternalEvent() intersvc.InternalEvent {
	triggers := make([]intersvc.Trigger, len(dto.Triggers))
	for i, t := range dto.Triggers {
		triggers[i] = t.ToInternalTrigger()
	}

	return intersvc.InternalEvent{
		Name:        dto.Name,
		ServiceName: dto.ServiceName,
		RepoUrl:     dto.RepoURL,
		TeamOwner:   dto.TeamOwner,
		Triggers:    intersvc.ListTrigger(triggers),
	}
}

type Trigger struct {
	ServiceName string            `json:"serviceName"`
	Type        string            `json:"type"`
	Host        string            `json:"host"`
	Path        string            `json:"path"`
	Headers     map[string]string `json:"headers"`
}

func (t Trigger) ToInternalTrigger() intersvc.Trigger {
	return intersvc.Trigger{
		ServiceName: t.ServiceName,
		Type:        intersvc.TriggerType(t.Type),
		BaseUrl:     t.Host,
		Path:        t.Path,
		Headers:     t.Headers,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		DeletedAt:   time.Time{},
	}
}

type CreateInternalEvent func(ctx context.Context, event intersvc.InternalEvent) error

func GetConsumerHandle(fn CreateInternalEvent) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var dto BulkEventDto
		if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		defer r.Body.Close()

		if err := fn(r.Context(), dto.ToInternalEvent()); err != nil {
			http.Error(w, "Failed to create internal event", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(dto); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}
