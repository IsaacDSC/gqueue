package pubadapter

import (
	"fmt"

	"github.com/hibiken/asynq"
)

type Opts struct {
	Attributes map[string]string
	AsynqOpts  []asynq.Option
	WQType     WQType
}

var EmptyOpts = Opts{
	Attributes: make(map[string]string),
	AsynqOpts:  []asynq.Option{},
}

type WQType string

func (wt WQType) Validate() error {
	switch wt {
	case LowThroughput, HighThroughput, LowLatency:
		return nil
	default:
		return fmt.Errorf("invalid WQType: %s", wt)
	}
}

const (
	LowThroughput  WQType = "low_throughput"
	HighThroughput WQType = "high_throughput"
	LowLatency     WQType = "low_latency"
)
