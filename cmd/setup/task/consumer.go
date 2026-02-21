package task

import (
	"context"
	"log"

	"github.com/IsaacDSC/gqueue/cmd/setup/middleware"
	"github.com/IsaacDSC/gqueue/internal/app/taskapp"
	"github.com/IsaacDSC/gqueue/internal/cfg"
	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/asynqsvc"
	"github.com/IsaacDSC/gqueue/pkg/topicutils"
	"github.com/hibiken/asynq"
)

func (s *Service) consumer(ctx context.Context, env cfg.Config, asynqCfg asynq.Config) {

	mux := asynq.NewServeMux()
	mux.Use(middleware.AsynqLogger)

	events := []asynqsvc.AsynqHandle{
		taskapp.GetRequestHandle(s.fetch, s.insightsStore).ToAsynqHandler(),
	}

	for _, event := range events {
		topic := topicutils.BuildTopicName(domain.ProjectID, event.TopicName)
		mux.HandleFunc(topic, event.Handler)
	}

	log.Println("[*] starting worker with configs")
	log.Println("[*] wq.concurrency", asynqCfg.Concurrency)
	log.Println("[*] Asynq Worker started. Press Ctrl+C to gracefully shutdown...")

	go func() {
		if err := s.asynqServer.Run(mux); err != nil {
			log.Printf("[!] Asynq server error: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	log.Println("[*] Context cancelled, initiating graceful shutdown...")
	s.asynqServer.Shutdown()
	log.Println("[*] Asynq server stopped gracefully")

}
