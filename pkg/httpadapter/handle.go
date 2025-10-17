package httpadapter

import "net/http"

type HttpHandle struct {
	Path    string
	Handler func(w http.ResponseWriter, r *http.Request)
}
