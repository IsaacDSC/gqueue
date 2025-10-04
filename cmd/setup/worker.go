package setup

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/IsaacDSC/gqueue/internal/cfg"
	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/internal/fetcher"
	"github.com/IsaacDSC/gqueue/pkg/asynqsvc"
	"github.com/IsaacDSC/gqueue/pkg/gpubsub"
	"github.com/IsaacDSC/gqueue/pkg/topicutils"

	"github.com/IsaacDSC/gqueue/internal/wtrhandler"
	"github.com/IsaacDSC/gqueue/pkg/cachemanager"
	"github.com/IsaacDSC/gqueue/pkg/publisher"

	"github.com/IsaacDSC/gqueue/internal/interstore"

	"github.com/hibiken/asynq"
)

func StartWorker(clientPubsub *pubsub.Client, cache cachemanager.Cache, store interstore.Repository, pub publisher.Publisher) {
	fetch := fetcher.NewNotification()

	cfg := cfg.Get()
	workerType := cfg.AsynqConfig.WorkerType

	fmt.Println("[*] worker type", workerType)
	if workerType == "googlepubsub" {
		startUsingGooglePubSub(clientPubsub, cache, store, pub, fetch)
	} else {
		startUsingAsynq(cache, store, pub, fetch)
	}

}

func startUsingGooglePubSub(clientPubsub *pubsub.Client, cache cachemanager.Cache, store interstore.Repository, pub publisher.Publisher, fetch *fetcher.Notification) {
	ctx := context.Background()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cfg := cfg.Get()

	queues := domain.GetTopics()
	cuncurrency := cfg.AsynqConfig.Concurrency

	handlers := []gpubsub.Handle{
		wtrhandler.NewDeadLatterQueue().ToGPubSubHandler(pub),
		wtrhandler.GetRequestHandle(fetch).ToGPubSubHandler(pub),
		wtrhandler.GetInternalConsumerHandle(store, cache, pub).ToGPubSubHandler(pub),
	}

	var wg sync.WaitGroup

	for _, handler := range handlers {
		wg.Add(1)
		go func(handler gpubsub.Handle) {
			defer wg.Done()

			topicName := topicutils.BuildTopicName(domain.ProjectID, handler.Event)
			log.Printf("[*] Starting subscriber for topic: %s", topicName)

			// Register topic if not exists
			topic := clientPubsub.Topic(topicName)
			exists, err := topic.Exists(ctx)
			if err != nil {
				log.Printf("[!] Error checking if topic %s exists: %v", topicName, err)
				return
			}

			if !exists {
				log.Printf("[*] Creating topic: %s", topicName)
				topic, err = clientPubsub.CreateTopic(ctx, topicName)
				if err != nil {
					log.Printf("[!] Error creating topic %s: %v", topicName, err)
					return
				}
			}

			subscriptionName := topicutils.BuildSubscriptionName(topicName)
			subscription := clientPubsub.Subscription(subscriptionName)

			subscription.Delete(ctx)

			subExists, err := subscription.Exists(ctx)
			if err != nil {
				log.Printf("[!] Error checking if subscription %s exists: %v", subscriptionName, err)
				return
			}

			if !subExists {
				log.Printf("[*] Creating subscription: %s", subscriptionName)

				subscription, err = clientPubsub.CreateSubscription(ctx, subscriptionName, pubsub.SubscriptionConfig{
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
				NumGoroutines:          cuncurrency,
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
	log.Println("[*] wq.concurrency", (len(queues)*len(handlers))*cuncurrency)
	log.Println("[*] wq.queues", queues)
	log.Println("[*] Worker started. Press Ctrl+C to gracefully shutdown...")

	<-sigChan
	log.Println("[*] Received shutdown signal, initiating graceful shutdown...")

	cancel()

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

func startUsingAsynq(cache cachemanager.Cache, store interstore.Repository, pub publisher.Publisher, fetch *fetcher.Notification) {
	cfg := cfg.Get()

	asyqCfg := asynq.Config{
		Concurrency: cfg.AsynqConfig.Concurrency,
		Queues:      cfg.AsynqConfig.Queues,
	}

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.Cache.CacheAddr},
		asyqCfg,
	)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	mux := asynq.NewServeMux()
	mux.Use(AsynqLogger)

	events := []asynqsvc.AsynqHandle{
		wtrhandler.GetRequestHandle(fetch).ToAsynqHandler(),
		wtrhandler.GetInternalConsumerHandle(store, cache, pub).ToAsynqHandler(),
	}

	for _, event := range events {
		mux.HandleFunc(event.Event, event.Handler)
	}

	log.Println("[*] starting worker with configs")
	log.Println("[*] wq.concurrency", asyqCfg.Concurrency)
	log.Println("[*] wq.queues", asyqCfg.Queues)
	log.Println("[*] Asynq Worker started. Press Ctrl+C to gracefully shutdown...")

	go func() {
		if err := srv.Run(mux); err != nil {
			log.Printf("[!] Asynq server error: %v", err)
		}
	}()

	<-sigChan
	log.Println("[*] Received shutdown signal, initiating graceful shutdown...")

	srv.Shutdown()
	log.Println("[*] Asynq server stopped gracefully")
}
