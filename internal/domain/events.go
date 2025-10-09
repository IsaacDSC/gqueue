package domain

const (
	EventQueueRequestToExternal = "event-queue.request-to-external"
	EventQueueInternal          = "event-queue.internal"
	EventQueueDeadLetter        = "event-queue.dead-letter"
)

func GetTopics() []string {
	return []string{
		EventQueueRequestToExternal,
		EventQueueInternal,
	}
}
