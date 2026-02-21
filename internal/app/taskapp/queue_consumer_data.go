package taskapp

type Consumer struct {
	Host    string            `json:"host"`
	Headers map[string]string `json:"headers"`
}

type QueueConsumers map[Queue][]Consumer
