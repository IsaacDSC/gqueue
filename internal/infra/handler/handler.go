package handler

import (
	"context"
	"net/http"

	"github.com/IsaacDSC/webhook/internal/eventqueue"

	"github.com/IsaacDSC/webhook/internal/intersvc"
	"github.com/IsaacDSC/webhook/internal/interweb"
	"github.com/IsaacDSC/webhook/pkg/publisher"
)

type Service interface {
	CreateInternalEvent(ctx context.Context, input intersvc.CreateInternalEventDto) (output intersvc.InternalEvent, err error)
	RegisterTrigger(ctx context.Context, input intersvc.RegisterTriggersDto) (output intersvc.InternalEvent, err error)
	CreateConsumer(ctx context.Context, event intersvc.InternalEvent) error
}

type Handler struct {
	service Service
	routes  map[string]func(http.ResponseWriter, *http.Request)
}

func NewHandler(service Service, pub publisher.Publisher) *Handler {
	h := &Handler{service: service}

	publisherHandle := eventqueue.Publisher(pub)
	h.routes = map[string]func(http.ResponseWriter, *http.Request){
		// Used for interface
		"POST /event/create":   interweb.GetCreateEventHandle(h.service.CreateInternalEvent),
		"POST /event/register": interweb.GetRegisterHandle(h.service.RegisterTrigger),

		// Used for cli
		"POST /event/consumer": interweb.GetConsumerHandle(h.service.CreateConsumer),

		// Used for SDK - CLI
		"POST /event/publisher": publisherHandle.Handler,
	}

	return h
}

func (h Handler) GetRoutes() map[string]func(http.ResponseWriter, *http.Request) {
	return h.routes
}
