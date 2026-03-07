package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type Metric struct {
	Name        string
	Description string
}

var (
	// Mem Store
	MemStoreEventNotFound = Metric{Name: "mem_store_event_not_found_total", Description: "Total of events not found in the memory store"}
	MemActivityDuration   = Metric{Name: "mem_store_activity_duration", Description: "Activity of the memory store"}

	// HTTP Server
	HTTPServerRequests        = Metric{Name: "http_server_requests_total", Description: "Total of requests to the HTTP server"} // Filter by http.response_code
	HTTPServerRequestDuration = Metric{Name: "http_server_request_duration_seconds", Description: "Duration of requests to the HTTP server"}
	// HTTP Client
	HTTPClientRequests        = Metric{Name: "http_client_requests_total", Description: "Total of requests to the HTTP client"}
	HTTPClientRequestDuration = Metric{Name: "http_client_request_duration_seconds", Description: "Duration of requests to the HTTP client"}
	// PubSub
	PubSubPublisherRequests  = Metric{Name: "pubsub_publisher_requests_total", Description: "Total of requests to the pubsub publisher"}
	PubSubConsumerRetries    = Metric{Name: "pubsub_consumer_retries_total", Description: "Total of retries for a consumer"}
	PubSubConsumerDlq        = Metric{Name: "pubsub_consumer_dlq_total", Description: "Total of archived messages for a consumer"}
	PubSubConsumerDuration   = Metric{Name: "pubsub_consumer_duration_seconds", Description: "Duration of a consumer"}
	PubSubConsumerLagSeconds = Metric{Name: "pubsub_consumer_lag_seconds", Description: "Time in seconds between message publish and consumer processing start"}
	//  Task
	TaskPublisherRequests       = Metric{Name: "task_publisher_requests_total", Description: "Total of requests to the task publisher"}
	TaskConsumerRetries         = Metric{Name: "task_consumer_retries_total", Description: "Total of retries for a consumer"}
	TaskConsumerArchived        = Metric{Name: "task_consumer_archived_total", Description: "Total of archived messages for a consumer"}
	TaskConsumerDuration        = Metric{Name: "task_consumer_duration_seconds", Description: "Duration of a consumer"}
	TaskConsumerLagSeconds      = Metric{Name: "task_consumer_lag_seconds", Description: "Time in seconds between message publish and consumer processing start"}
	TaskConsumerTotalProcessing = Metric{Name: "task_consumer_total_processing", Description: "Total of tasks being consumed"}           // Filter by task.event_name
	TaskConsumerTotalFailure    = Metric{Name: "task_consumer_total_failure", Description: "Total of tasks being consumed with failure"} // Filter by task.event_name
	TaskConsumerTotalSuccess    = Metric{Name: "task_consumer_total_success", Description: "Total of tasks being consumed with success"} // Filter by task.event_name
)

func (m Metric) Count(ctx context.Context, value int64, attrs ...attribute.KeyValue) {
	meter := MeterFromContext(ctx)
	if counter, err := meter.Int64Counter(m.Name, metric.WithDescription(m.Description)); err == nil {
		counter.Add(ctx, value, metric.WithAttributes(attrs...))
	}
}

func (m Metric) Increment(ctx context.Context, attrs ...attribute.KeyValue) {
	meter := MeterFromContext(ctx)
	if upDown, err := meter.Int64UpDownCounter(m.Name, metric.WithDescription(m.Description)); err == nil {
		upDown.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

func (m Metric) Decrement(ctx context.Context, attrs ...attribute.KeyValue) {
	meter := MeterFromContext(ctx)
	if upDown, err := meter.Int64UpDownCounter(m.Name, metric.WithDescription(m.Description)); err == nil {
		upDown.Add(ctx, -1, metric.WithAttributes(attrs...))
	}
}

func (m Metric) Record(ctx context.Context, value float64, attrs ...attribute.KeyValue) {
	meter := MeterFromContext(ctx)
	if histogram, err := meter.Float64Histogram(m.Name, metric.WithDescription(m.Description)); err == nil {
		histogram.Record(ctx, value, metric.WithAttributes(attrs...))
	}
}
