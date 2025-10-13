package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInsights(t *testing.T) {
	t.Run("Given metrics with success and error cases, when transform to insights, then should calculate correctly", func(t *testing.T) {
		baseTime := time.Date(2025, 10, 11, 10, 7, 47, 0, time.UTC)

		metrics := Metrics{
			// Published metrics (ConsumerName empty) - 2 success, 1 error
			{
				TopicName:      "payment.processed",
				ConsumerName:   "",
				TimeStarted:    baseTime,
				TimeEnded:      baseTime.Add(20 * time.Millisecond),
				TimeDurationMs: 20,
				ACK:            true,
			},
			{
				TopicName:      "payment.processed",
				ConsumerName:   "",
				TimeStarted:    baseTime.Add(1 * time.Minute),
				TimeEnded:      baseTime.Add(1*time.Minute + 15*time.Millisecond),
				TimeDurationMs: 15,
				ACK:            true,
			},
			{
				TopicName:      "payment.processed",
				ConsumerName:   "",
				TimeStarted:    baseTime.Add(2 * time.Minute),
				TimeEnded:      baseTime.Add(2*time.Minute + 10*time.Millisecond),
				TimeDurationMs: 10,
				ACK:            false, // Error case
			},
			{
				TopicName:      "order.created",
				ConsumerName:   "",
				TimeStarted:    baseTime,
				TimeEnded:      baseTime.Add(25 * time.Millisecond),
				TimeDurationMs: 25,
				ACK:            true,
			},
			// Consumed metrics (ConsumerName not empty) - 3 success, 1 error
			{
				TopicName:      "payment.processed",
				ConsumerName:   "consumer-1",
				TimeStarted:    baseTime,
				TimeEnded:      baseTime.Add(5 * time.Millisecond),
				TimeDurationMs: 5,
				ACK:            true,
			},
			{
				TopicName:      "payment.processed",
				ConsumerName:   "consumer-1",
				TimeStarted:    baseTime.Add(1 * time.Minute),
				TimeEnded:      baseTime.Add(1*time.Minute + 3*time.Millisecond),
				TimeDurationMs: 3,
				ACK:            true,
			},
			{
				TopicName:      "payment.processed",
				ConsumerName:   "consumer-2",
				TimeStarted:    baseTime,
				TimeEnded:      baseTime.Add(8 * time.Millisecond),
				TimeDurationMs: 8,
				ACK:            true,
			},
			{
				TopicName:      "payment.processed",
				ConsumerName:   "consumer-2",
				TimeStarted:    baseTime.Add(1 * time.Minute),
				TimeEnded:      baseTime.Add(1*time.Minute + 12*time.Millisecond),
				TimeDurationMs: 12,
				ACK:            false, // Error case
			},
		}

		insights := metrics.Insights()

		expected := Insights{
			TotalPublished:             4,
			TotalConsumed:              4,
			TotalPublishedWithSuccess:  3,
			TotalConsumedWithSuccess:   3,
			TotalPublishedWithErr:      1,
			TotalConsumedWithErr:       1,
			PercentagePublishedSuccess: "75.00%",
			PercentageConsumedSuccess:  "75.00%",
			TotalSegmentationPublished: map[string]int64{
				"payment.processed": 3,
				"order.created":     1,
			},
			TotalSegmentationConsumed: map[string]int64{
				"consumer-1": 2,
				"consumer-2": 2,
			},
			RpmPublisher: map[string]*RPM{
				"payment.processed": {
					Timeseries: []time.Time{
						time.Date(2025, 10, 11, 10, 7, 0, 0, time.UTC),
						time.Date(2025, 10, 11, 10, 8, 0, 0, time.UTC),
						time.Date(2025, 10, 11, 10, 9, 0, 0, time.UTC),
					},
					Values: []int64{1, 1, 1},
				},
				"order.created": {
					Timeseries: []time.Time{
						time.Date(2025, 10, 11, 10, 7, 0, 0, time.UTC),
					},
					Values: []int64{1},
				},
			},
			RpmConsumer: map[string]*RPM{
				"payment.processed:consumer-1": {
					Timeseries: []time.Time{
						time.Date(2025, 10, 11, 10, 7, 0, 0, time.UTC),
						time.Date(2025, 10, 11, 10, 8, 0, 0, time.UTC),
					},
					Values: []int64{1, 1},
				},
				"payment.processed:consumer-2": {
					Timeseries: []time.Time{
						time.Date(2025, 10, 11, 10, 7, 0, 0, time.UTC),
						time.Date(2025, 10, 11, 10, 8, 0, 0, time.UTC),
					},
					Values: []int64{1, 1},
				},
			},
			P99Consumed: map[string]float64{
				"payment.processed.consumer-1": 5.0, // max of [5, 3]
				"payment.processed.consumer-2": 8.0, // max of [8, 12]
			},
			P75Consumed: map[string]float64{
				"payment.processed.consumer-1": 5.0, // 75th percentile of [5, 3]
				"payment.processed.consumer-2": 8.0, // 75th percentile of [8, 12]
			},
			P99Published: map[string]float64{
				"payment.processed": 20.0, // max of [20, 15, 10]
				"order.created":     25.0, // only one value [25]
			},
			P75Published: map[string]float64{
				"payment.processed": 20.0, // 75th percentile of [20, 15, 10]
				"order.created":     25.0, // only one value [25]
			},
		}

		assert.Equal(t, expected.TotalPublished, insights.TotalPublished)
		assert.Equal(t, expected.TotalConsumed, insights.TotalConsumed)
		assert.Equal(t, expected.TotalPublishedWithSuccess, insights.TotalPublishedWithSuccess)
		assert.Equal(t, expected.TotalConsumedWithSuccess, insights.TotalConsumedWithSuccess)
		assert.Equal(t, expected.TotalPublishedWithErr, insights.TotalPublishedWithErr)
		assert.Equal(t, expected.TotalConsumedWithErr, insights.TotalConsumedWithErr)
		assert.Equal(t, expected.PercentagePublishedSuccess, insights.PercentagePublishedSuccess)
		assert.Equal(t, expected.PercentageConsumedSuccess, insights.PercentageConsumedSuccess)
		assert.Equal(t, expected.TotalSegmentationPublished, insights.TotalSegmentationPublished)
		assert.Equal(t, expected.TotalSegmentationConsumed, insights.TotalSegmentationConsumed)
		assert.Equal(t, expected.RpmPublisher, insights.RpmPublisher)
		assert.Equal(t, expected.RpmConsumer, insights.RpmConsumer)
		assert.Equal(t, expected.P99Consumed, insights.P99Consumed)
		assert.Equal(t, expected.P75Consumed, insights.P75Consumed)
		assert.Equal(t, expected.P99Published, insights.P99Published)
		assert.Equal(t, expected.P75Published, insights.P75Published)
	})
}
