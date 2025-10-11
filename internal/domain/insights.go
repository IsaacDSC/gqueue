package domain

import (
	"strings"
	"time"
)

type ConsumerMetric struct {
	TopicName      string
	ConsumerName   string
	TimeStarted    time.Time
	TimeEnded      time.Time
	TimeDurationMs int64
	ACK            bool
}

type PublisherMetric struct {
	TopicName      string
	TimeStarted    time.Time
	TimeEnded      time.Time
	TimeDurationMs int64
	ACK            bool
}

type Metric struct {
	TopicName      string
	ConsumerName   string
	TimeStarted    time.Time
	TimeEnded      time.Time
	TimeDurationMs int64
	ACK            bool
}

type Metrics []Metric

func (m *Metrics) Insights() Insights {
	insights := NewInsights()

	for _, metric := range *m {
		dateInMinute := time.Date(metric.TimeEnded.Year(), metric.TimeEnded.Month(), metric.TimeEnded.Day(), metric.TimeEnded.Hour(), metric.TimeEnded.Minute(), 0, 0, time.UTC)

		if metric.ConsumerName != "" {
			if metric.ACK {
				insights.TotalConsumed++
			} else {
				insights.TotalConsumedWithErr++
			}

			insights.TotalSegmentationConsumed[metric.ConsumerName]++
			insights.SegmentationConsumed.add(metric)
			insights.RpmConsumer.add(metric.TopicName, metric.ConsumerName, dateInMinute)
			continue

		} else {
			if metric.ACK {
				insights.TotalPublished++
			} else {
				insights.TotalPublishedWithErr++
			}

			insights.TotalSegmentationPublished[metric.TopicName]++
			insights.SegmentationPublished.add(metric)
			insights.RpmPublisher.add(metric.TopicName, dateInMinute)
		}
	}

	return insights
}

type Insights struct {
	TotalPublished             int64                 `json:"total_published"`
	TotalConsumed              int64                 `json:"total_consumed"`
	TotalPublishedWithErr      int64                 `json:"total_published_with_err"`
	TotalConsumedWithErr       int64                 `json:"total_consumed_with_err"`
	TotalSegmentationPublished map[string]int64      `json:"total_segmentation_published"`
	TotalSegmentationConsumed  map[string]int64      `json:"total_segmentation_consumed"`
	SegmentationPublished      SegmentationPublished `json:"segmentation_published"`
	SegmentationConsumed       SegmentationConsumed  `json:"segmentation_consumed"`
	RpmPublisher               RpmPublisher          `json:"rpm_publisher"`
	RpmConsumer                RpmConsumer           `json:"rpm_consumer"`
}

func NewInsights() Insights {
	return Insights{
		TotalPublished:             0,
		TotalConsumed:              0,
		TotalSegmentationPublished: make(map[string]int64),
		TotalSegmentationConsumed:  make(map[string]int64),
		SegmentationPublished:      make(map[string][]PublisherMetric),
		SegmentationConsumed:       make(map[string][]ConsumerMetric),
		RpmPublisher:               make(map[string]map[time.Time]int64),
		RpmConsumer:                make(map[string]map[time.Time]int64),
	}
}

type RpmPublisher map[string]map[time.Time]int64

func (rp RpmPublisher) add(topicName string, dateInMinute time.Time) {
	if data, ok := rp[topicName]; ok {
		data[dateInMinute]++
	} else {
		rp[topicName] = map[time.Time]int64{dateInMinute: 1}
	}
}

type RpmConsumer map[string]map[time.Time]int64

func (rc RpmConsumer) add(topicName, consumerName string, dateInMinute time.Time) {
	key := strings.Join([]string{topicName, consumerName}, ":")
	if data, ok := rc[key]; ok {
		data[dateInMinute]++
	} else {
		rc[key] = map[time.Time]int64{dateInMinute: 1}
	}
}

type SegmentationPublished map[string][]PublisherMetric

func (sp SegmentationPublished) add(metric Metric) {
	key := metric.TopicName
	sp[key] = append(sp[key], PublisherMetric{
		TopicName:      metric.TopicName,
		TimeStarted:    metric.TimeStarted,
		TimeEnded:      metric.TimeEnded,
		TimeDurationMs: metric.TimeDurationMs,
		ACK:            metric.ACK,
	})
}

type SegmentationConsumed map[string][]ConsumerMetric

func (sc SegmentationConsumed) add(metric Metric) {
	key := strings.Join([]string{metric.TopicName, metric.ConsumerName}, ".")
	sc[key] = append(sc[key], ConsumerMetric(metric))
}
