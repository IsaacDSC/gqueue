package notifyopt

type Kind string

func (k Kind) String() string {
	return string(k)
}

const (
	Default        Kind = "default"
	HighThroughput Kind = "high_throughput"
	LongRunning    Kind = "long_running"
)
