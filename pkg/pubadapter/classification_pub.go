package pubadapter

type ClassificationResult struct {
	InternalPublisher GenericPublisher
	ExternalPublisher GenericPublisher
}

func ClassificationPublisher(gcppubsub, redisAsync GenericPublisher) ClassificationResult {
	return ClassificationResult{
		InternalPublisher: redisAsync,
		ExternalPublisher: gcppubsub,
	}
}
