package pubadapter

import (
	"fmt"
)

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
