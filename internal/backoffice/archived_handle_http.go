package backoffice

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/IsaacDSC/gqueue/internal/task"
	"github.com/IsaacDSC/gqueue/pkg/httpsvc"
)

type TaskArchiver interface {
	GetAllMsgsByQueue(ctx context.Context, queue string) ([]task.TaskArchivedData, error)
}

// asynq:{external.medium}:archived
func TaskArchivedHandle(taskArchived TaskArchiver) httpsvc.HttpHandle {
	return httpsvc.HttpHandle{
		Path: "GET /tasks/archived/{queue_name}",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			queueName := r.PathValue("queue")

			msgs, err := taskArchived.GetAllMsgsByQueue(ctx, queueName)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(msgs); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		},
	}
}
