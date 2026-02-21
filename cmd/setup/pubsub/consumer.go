package pubsub

import (
	"context"
	"log"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/IsaacDSC/gqueue/internal/app/pubsubapp"
	"github.com/IsaacDSC/gqueue/internal/cfg"
	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/gpubsub"
	"github.com/IsaacDSC/gqueue/pkg/topicutils"
)

func (s *Service) consumer(ctx context.Context, env cfg.Config) {
	// Use only the context passed from main for shutdown control

	concurrency := env.AsynqConfig.Concurrency

	handlers := []gpubsub.Handle{
		pubsubapp.NewDeadLatterQueue(s.memStore, s.fetch).ToGPubSubHandler(s.gcppublisher),
		pubsubapp.GetRequestHandle(s.fetch, s.insightsStore).ToGPubSubHandler(s.gcppublisher),
	}

	var wg sync.WaitGroup

	for _, handler := range handlers {
		wg.Add(1)
		go func(handler gpubsub.Handle) {
			defer wg.Done()

			topicName := topicutils.BuildTopicName(domain.ProjectID, handler.TopicName)
			log.Printf("[*] Starting subscriber for topic: %s", topicName)

			// Register topic if not exists
			topic := s.pubsubClient.Topic(topicName)
			exists, err := topic.Exists(ctx)
			if err != nil {
				log.Printf("[!] Error checking if topic %s exists: %v", topicName, err)
				return
			}

			if !exists {
				log.Printf("[*] Creating topic: %s", topicName)
				topic, err = s.pubsubClient.CreateTopic(ctx, topicName)
				if err != nil {
					log.Printf("[!] Error creating topic %s: %v", topicName, err)
					return
				}
			}

			subscriptionName := topicutils.BuildSubscriptionName(topicName)
			subscription := s.pubsubClient.Subscription(subscriptionName)

			subExists, err := subscription.Exists(ctx)
			if err != nil {
				log.Printf("[!] Error checking if subscription %s exists: %v", subscriptionName, err)
				return
			}

			if !subExists {
				log.Printf("[*] Creating subscription: %s", subscriptionName)

				subscription, err = s.pubsubClient.CreateSubscription(ctx, subscriptionName, pubsub.SubscriptionConfig{
					Topic:       topic,
					AckDeadline: 20 * time.Second,
					// DeadLetterPolicy: &pubsub.DeadLetterPolicy{
					// 	DeadLetterTopic:     topicutils.BuildTopicName(domain.ProjectID, domain.EventQueueDeadLatter),
					// 	MaxDeliveryAttempts: 10,
					// },
				})
				if err != nil {
					log.Printf("[!] Error creating subscription %s: %v", subscriptionName, err)
					return
				}
			}

			subscription.ReceiveSettings = pubsub.ReceiveSettings{
				MaxExtension:           60 * time.Minute,
				MaxOutstandingMessages: 1000,
				MaxOutstandingBytes:    1e9,
				NumGoroutines:          concurrency,
			}

			if err := subscription.Receive(ctx, handler.Handler); err != nil {
				if ctx.Err() == context.Canceled {
					log.Printf("[*] Subscriber for topic %s shutting down gracefully", topicName)
				} else {
					log.Printf("[!] Error in subscriber for topic %s: %v", topicName, err)
				}
			}

		}(handler)
	}

	log.Println("[*] starting worker with configs")
	log.Println("[*] wq.concurrency", (len(handlers))*concurrency)
	log.Println("[*] Worker started. Waiting for shutdown signal from context...")

	<-ctx.Done()
	log.Println("[*] Context cancelled, initiating graceful shutdown...")

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("[*] All subscribers stopped gracefully")
	case <-time.After(30 * time.Second):
		log.Println("[!] Timeout waiting for subscribers to stop, forcing shutdown")
	}
}
