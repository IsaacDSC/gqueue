package asyncadapter

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/asynqsvc"
	"github.com/IsaacDSC/gqueue/pkg/gpubsub"
	"github.com/IsaacDSC/gqueue/pkg/publisher"
	"github.com/IsaacDSC/gqueue/pkg/topicutils"
	"github.com/hibiken/asynq"
)

type AdapterType string

type AsyncCtx[T any] struct {
	ctx         context.Context
	payload     T
	bytePayload []byte
}

func (c AsyncCtx[T]) Bytes() []byte {
	return c.bytePayload
}

func (c AsyncCtx[T]) Payload() (T, error) {
	var empty T

	if err := json.Unmarshal(c.bytePayload, &c.payload); err != nil {
		return empty, fmt.Errorf("unmarshal payload: %w", err)
	}

	return c.payload, nil
}

func (c AsyncCtx[T]) Context() context.Context {
	return c.ctx
}

type Handle[T any] struct {
	Event   string
	Handler func(c AsyncCtx[T]) error
}

func (h Handle[T]) ToAsynqHandler() asynqsvc.AsynqHandle {
	return asynqsvc.AsynqHandle{
		Event: h.Event,
		Handler: func(ctx context.Context, task *asynq.Task) error {
			if err := h.Handler(AsyncCtx[T]{
				ctx:         ctx,
				bytePayload: task.Payload(),
			}); err != nil {
				return fmt.Errorf("handle task: %w", err)
			}

			return nil
		},
	}
}

func (h Handle[T]) ToGPubSubHandler(pub publisher.Publisher) gpubsub.Handle {

	archivedMsg := func(ctx context.Context, msg *pubsub.Message) {
		defer msg.Ack()
		time.Sleep(time.Second * 5)
		topic := topicutils.BuildTopicName(domain.ProjectID, domain.EventQueueDeadLetter)
		if err := pub.Publish(ctx, topic, msg, publisher.Opts{
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

		strMaxRetryCount := msg.Attributes["max_attempts"]

		retryCount, err := strconv.Atoi(strRetryCount)
		if err != nil {
			panic(err)
		}

		maxRetryAttempts, err := strconv.Atoi(strMaxRetryCount)
		if err != nil {
			panic(err)
		}

		if retryCount >= maxRetryAttempts {
			archivedMsg(ctx, msg)
			return
		}

		retryCount++
		msg.Attributes["retry_count"] = strconv.Itoa(retryCount)
		topic := msg.Attributes["topic"]
		time.Sleep(time.Second * 5)
		if err := pub.Publish(ctx, topic, msg, publisher.Opts{
			Attributes: msg.Attributes,
		}); err != nil {
			msg.Nack()
			return
		}

	}

	return gpubsub.Handle{
		Event: h.Event,
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
