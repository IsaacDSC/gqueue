package asyncadapter

import (
	"context"
	"strconv"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/gpubsub"
	"github.com/IsaacDSC/gqueue/pkg/pubadapter"
	"github.com/IsaacDSC/gqueue/pkg/topicutils"
)

func (h Handle[T]) ToGPubSubHandler(pub pubadapter.GenericPublisher) gpubsub.Handle {

	archivedMsg := func(ctx context.Context, msg *pubsub.Message) {
		defer msg.Ack()
		topic := topicutils.BuildTopicName(domain.ProjectID, domain.EventQueueDeadLetter)
		if err := pub.Publish(ctx, topic, msg, pubadapter.Opts{
			Attributes: msg.Attributes,
		}); err != nil {
			msg.Nack()
		}
	}

	retryable := func(ctx context.Context, msg *pubsub.Message) {
		defer msg.Ack()

		strRetryCount, ok := msg.Attributes["retry_count"]
		if !ok {
			strRetryCount = "0"
		}

		strMaxRetryCount := msg.Attributes["max_retries"]

		retryCount, err := strconv.Atoi(strRetryCount)
		if err != nil {
			panic(err) //TODO: add better validation and error handling
		}

		maxRetryAttempts, err := strconv.Atoi(strMaxRetryCount)
		if err != nil {
			panic(err) //TODO: add better validation and error handling
		}

		if retryCount >= maxRetryAttempts {
			archivedMsg(ctx, msg)
			return
		}

		retryCount++
		msg.Attributes["retry_count"] = strconv.Itoa(retryCount)
		topic := msg.Attributes["topic"]

		// Wait respecting the context
		select {
		case <-time.After(5 * time.Second):
			// continue
		case <-ctx.Done():
			return
		}

		if ctx.Err() != nil {
			return
		}

		if err := pub.Publish(ctx, topic, msg, pubadapter.Opts{
			Attributes: msg.Attributes,
		}); err != nil {
			msg.Nack()
			return
		}
	}

	return gpubsub.Handle{
		TopicName: h.EventName,
		Handler: func(ctx context.Context, msg *pubsub.Message) {
			defer msg.Nack()

			if err := h.Handler(AsyncCtx[T]{
				ctx:         ctx,
				bytePayload: msg.Data,
			}); err != nil {
				msg.Attributes["msg"] = err.Error()
				retryable(ctx, msg)
				return
			}

			msg.Ack()
		},
	}
}
