package pubadapter

type ClassificationResult struct {
	InternalPublisher          GenericPublisher
	ExternalPublisher          GenericPublisher
	HighPerformancePublisher   GenericPublisher
	MediumPerformancePublisher GenericPublisher
}

func ClassificationPublisher(gcppubsub, redisAsync GenericPublisher) ClassificationResult {
	return ClassificationResult{
		InternalPublisher:          redisAsync,
		MediumPerformancePublisher: redisAsync,
		ExternalPublisher:          gcppubsub,
		HighPerformancePublisher:   gcppubsub,
	}
}
