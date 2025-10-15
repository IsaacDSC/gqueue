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

func (h Handle[T]) ToGPubSubHandler(pub pubadapter.Publisher) gpubsub.Handle {

	archivedMsg := func(ctx context.Context, msg *pubsub.Message) {
		defer msg.Ack()
		topic := topicutils.BuildTopicName(domain.ProjectID, domain.EventQueueDeadLetter)
		if err := pub.Publish(ctx, pubadapter.HighThroughput, topic, msg, pubadapter.Opts{
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
			panic(err) //TODO: adicionar validação e tratamento de erro melhor
		}

		maxRetryAttempts, err := strconv.Atoi(strMaxRetryCount)
		if err != nil {
			panic(err) //TODO: adicionar validação e tratamento de erro melhor
		}

		if retryCount >= maxRetryAttempts {
			archivedMsg(ctx, msg)
			return
		}

		retryCount++
		msg.Attributes["retry_count"] = strconv.Itoa(retryCount)
		topic := msg.Attributes["topic"]
		time.Sleep(time.Second * 5)
		if err := pub.Publish(ctx, pubadapter.HighThroughput, topic, msg, pubadapter.Opts{
			Attributes: msg.Attributes,
		}); err != nil {
			msg.Nack()
			return
		}

	}

	return gpubsub.Handle{
		TopicName: h.TopicName,
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
