package domain

const (
	EventQueueRequestToExternal = "event-queue.request-to-external"
	EventQueueInternal          = "event-queue.internal"
	EventQueueDeadLatter        = "event-queue.dead-latter"
)

func GetTopics() []string {
	return []string{
		EventQueueRequestToExternal,
		EventQueueInternal,
	}
}
