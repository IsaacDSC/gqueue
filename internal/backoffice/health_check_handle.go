package backoffice

import (
	"net/http"

	"github.com/IsaacDSC/gqueue/pkg/httpsvc"
)

func GetHealthCheckHandler() httpsvc.HttpHandle {
	return httpsvc.HttpHandle{
		Path: "GET /api/v1/ping",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("pong"))
		},
	}
}
