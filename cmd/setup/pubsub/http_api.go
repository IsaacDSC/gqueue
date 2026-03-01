package pubsub

import (
	"context"
	"net/http"

	"github.com/IsaacDSC/gqueue/cmd/setup/httpsvc"
	"github.com/IsaacDSC/gqueue/internal/app/pubsubapp"
	"github.com/IsaacDSC/gqueue/internal/cfg"
	"github.com/IsaacDSC/gqueue/pkg/httpadapter"
)

func (s *Service) startHttpServer(ctx context.Context, env cfg.Config) *http.Server {
	routes := []httpadapter.HttpHandle{
		pubsubapp.PublisherEvent(s.memStore, s.gcppublisher, s.insightsStore),
	}

	return httpsvc.StartHttpServer(ctx, env, routes, env.PubsubApiPort.String())
}
