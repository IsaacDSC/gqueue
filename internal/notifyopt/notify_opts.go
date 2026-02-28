package notifyopt

type Kind string

const (
	Default        Kind = "default"
	HighThroughput Kind = "high_throughput"
	LongRunning    Kind = "long_running"
)
