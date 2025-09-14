package task

type Consumer struct {
	Host    string            `json:"host"`
	Headers map[string]string `json:"headers"`
}

type QueueConsumers map[Queue][]Consumer

func (q QueueConsumers) RmNotContains(listQueues Queues) {
	for queue := range q {
		if !listQueues.Contains(queue) {
			delete(q, queue)
		}
	}

}
