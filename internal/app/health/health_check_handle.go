package health

import (
	"net/http"

	"github.com/IsaacDSC/gqueue/pkg/httpadapter"
)

func GetHealthCheckHandler() httpadapter.HttpHandle {
	return httpadapter.HttpHandle{
		Path: "GET /api/v1/ping",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("pong"))
		},
	}
}
