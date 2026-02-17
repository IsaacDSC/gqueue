package wtrhandler

import (
	"encoding/json"
	"net/http"

	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/httpadapter"
	"github.com/IsaacDSC/gqueue/pkg/pubadapter"
	"github.com/IsaacDSC/gqueue/pkg/topicutils"
)

func Publisher(pub pubadapter.Publisher) httpadapter.HttpHandle {
	return httpadapter.HttpHandle{
		Path: "POST /api/v1/event/publisher",
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

			if payload.Opts.WqType == "" || payload.Opts.ScheduleIn > 0 {
				payload.Opts.WqType = pubadapter.LowThroughput
			}

			topic := topicutils.BuildTopicName(domain.ProjectID, domain.EventQueueInternal)
			opts := pubadapter.Opts{Attributes: payload.Attributes(topic), AsynqOpts: payload.AsynqOpts()}
			if err := pub.Publish(r.Context(), payload.Opts.WqType, topic, payload, opts); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusAccepted)
		},
	}
}
