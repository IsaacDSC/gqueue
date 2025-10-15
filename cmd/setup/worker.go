package setup

import (
	"context"
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
	"github.com/IsaacDSC/gqueue/internal/storests"
	"github.com/IsaacDSC/gqueue/pkg/asynqsvc"
	"github.com/IsaacDSC/gqueue/pkg/gpubsub"
	"github.com/IsaacDSC/gqueue/pkg/topicutils"

	"github.com/IsaacDSC/gqueue/internal/wtrhandler"
	"github.com/IsaacDSC/gqueue/pkg/cachemanager"
	"github.com/IsaacDSC/gqueue/pkg/pubadapter"

	"github.com/IsaacDSC/gqueue/internal/interstore"

	"github.com/hibiken/asynq"
)

type Worker struct {
	clientPubsub *pubsub.Client
}

func NewWorker() *Worker {
	return &Worker{}
}

func (w *Worker) WithClientPubsub(clientPubsub *pubsub.Client) *Worker {
	w.clientPubsub = clientPubsub
	return w
}

func (w *Worker) Start(cache cachemanager.Cache, store interstore.Repository, redisAsync pubadapter.Publisher, insightsStore *storests.Store) {
	fetch := fetcher.NewNotification()

	if w.clientPubsub != nil {
		go startUsingGooglePubSub(w.clientPubsub, cache, store, redisAsync, fetch, insightsStore)
	}

	startUsingAsynq(cache, store, redisAsync, fetch, insightsStore)
}

func startUsingGooglePubSub(clientPubsub *pubsub.Client, cache cachemanager.Cache, store interstore.Repository, pub pubadapter.Publisher, fetch *fetcher.Notification, insightsStore *storests.Store) {
	ctx := context.Background()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cfg := cfg.Get()

	concurrency := cfg.AsynqConfig.Concurrency

	handlers := []gpubsub.Handle{
		wtrhandler.NewDeadLatterQueue(store, fetch).ToGPubSubHandler(pub),
		wtrhandler.GetRequestHandle(fetch, insightsStore).ToGPubSubHandler(pub),
		wtrhandler.GetInternalConsumerHandle(store, cache, pub, insightsStore).ToGPubSubHandler(pub),
	}

	var wg sync.WaitGroup

	for _, handler := range handlers {
		wg.Add(1)
		go func(handler gpubsub.Handle) {
			defer wg.Done()

			topicName := topicutils.BuildTopicName(domain.ProjectID, handler.TopicName)
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

func startUsingAsynq(cache cachemanager.Cache, store interstore.Repository, pub pubadapter.Publisher, fetch *fetcher.Notification, insightsStore *storests.Store) {
	cfg := cfg.Get()

	asynqCfg := asynq.Config{
		Concurrency: cfg.AsynqConfig.Concurrency,
	}

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.Cache.CacheAddr},
		asynqCfg,
	)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	mux := asynq.NewServeMux()
	mux.Use(AsynqLogger)

	events := []asynqsvc.AsynqHandle{
		wtrhandler.GetRequestHandle(fetch, insightsStore).ToAsynqHandler(),
		wtrhandler.GetInternalConsumerHandle(store, cache, pub, insightsStore).ToAsynqHandler(),
	}

	for _, event := range events {
		topic := topicutils.BuildTopicName(domain.ProjectID, event.TopicName)
		mux.HandleFunc(topic, event.Handler)
	}

	log.Println("[*] starting worker with configs")
	log.Println("[*] wq.concurrency", asynqCfg.Concurrency)
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
