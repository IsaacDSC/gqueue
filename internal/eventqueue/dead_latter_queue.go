package eventqueue

import (
	"log"

	"cloud.google.com/go/pubsub"
	"github.com/IsaacDSC/gqueue/internal/domain"
	"github.com/IsaacDSC/gqueue/pkg/asyncadapter"
)

// TODO: implement logic
func NewDeadLatterQueue() asyncadapter.Handle[pubsub.Message] {
	return asyncadapter.Handle[pubsub.Message]{
		Event: domain.EventQueueDeadLatter,
		Handler: func(c asyncadapter.AsyncCtx[pubsub.Message]) error {
			log.Println("[*] DeadLatterQueue, received msg")

			p, err := c.Payload()
			if err != nil {
				return err
			}

			log.Println("DeadLatterQueue, payload:", p.ID)

			return nil
		},
	}
}
