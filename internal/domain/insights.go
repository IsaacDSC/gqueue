package domain

import "time"

type ConsumerInsights struct {
	TopicName    string
	ConsumerName string
	TimeStarted  time.Time
	TimeEnded    time.Time
	TimeDuration time.Duration
	ACK          bool
}

type PublisherInsights struct {
	TopicName    string
	TimeStarted  time.Time
	TimeEnded    time.Time
	TimeDuration time.Duration
	ACK          bool
}
