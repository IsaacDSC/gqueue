package pubadapter

import (
	"context"
	"fmt"

	"github.com/IsaacDSC/gqueue/internal/cfg"
)

type WQType string

func (wt WQType) Validate() error {
	switch wt {
	case Internal, External, LowThroughput, HighThroughput, LowLatency:
		return nil
	default:
		return fmt.Errorf("invalid WQType: %s", wt)
	}
}

const (
	Internal       WQType = "internal"
	External       WQType = "external"
	LowThroughput  WQType = "low_throughput"
	HighThroughput WQType = "high_throughput"
	LowLatency     WQType = "low_latency"
)

var wqMapper = map[WQType]cfg.WQ{
	Internal:       cfg.WQRedis,
	External:       cfg.WQGooglePubSub,
	LowThroughput:  cfg.WQRedis,
	HighThroughput: cfg.WQGooglePubSub,
	LowLatency:     cfg.WQRedis,
}

type Pub struct {
	gps    GenericPublisher
	rdp    GenericPublisher
	cfg    map[WQType]cfg.WQ
	wqType cfg.WQ
}

func NewPub(
	highPerformance GenericPublisher,
	mediumPerformance GenericPublisher,
	wQType cfg.WQ,
) *Pub {
	return &Pub{
		gps:    highPerformance,
		rdp:    mediumPerformance,
		cfg:    wqMapper,
		wqType: wQType,
	}
}

func (p *Pub) Publish(ctx context.Context, wqtype WQType, topicName string, payload any, opts Opts) error {
	if p.wqType == cfg.WQRedis {
		return p.rdp.Publish(ctx, topicName, payload, opts)
	}

	wq := p.cfg[wqtype]
	switch wq {
	case cfg.WQGooglePubSub:
		return p.gps.Publish(ctx, topicName, payload, opts)
	default:
		return p.rdp.Publish(ctx, topicName, payload, opts)
	}

}
