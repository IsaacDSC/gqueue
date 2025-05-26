package handler

import (
	"github.com/IsaacDSC/webhook/internal/service"
	"net/http"
)

type Handler struct {
	service service.Service
}

func NewHandler(service service.Service) *Handler {
	return &Handler{service: service}
}

func (h Handler) GetRoutes() map[string]func(http.ResponseWriter, *http.Request) {
	return map[string]func(http.ResponseWriter, *http.Request){
		"POST /event/create":    h.CreateInternalEvent,
		"POST /event/register":  h.RegisterTrigger,
		"POST /event/publisher": h.PublisherExternalEvent,
	}
}
