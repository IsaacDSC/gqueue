package domain

import (
	"fmt"
	"math"
	"sort"
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
	var (
		timeDurationsMsConsumed  = make(map[string][]float64)
		timeDurationsMsPublished = make(map[string][]float64)
	)

	for _, metric := range *m {
		dateInMinute := time.Date(metric.TimeEnded.Year(), metric.TimeEnded.Month(), metric.TimeEnded.Day(), metric.TimeEnded.Hour(), metric.TimeEnded.Minute(), 0, 0, time.UTC)

		if metric.ConsumerName != "" {
			insights.TotalConsumed++
			if metric.ACK {
				insights.TotalConsumedWithSuccess++
			} else {
				insights.TotalConsumedWithErr++
			}

			key := strings.Join([]string{metric.TopicName, metric.ConsumerName}, ".")
			df, ok := timeDurationsMsConsumed[key]
			if ok {
				df = append(df, float64(metric.TimeDurationMs))
			} else {
				timeDurationsMsConsumed[key] = []float64{float64(metric.TimeDurationMs)}
			}

			insights.TotalSegmentationConsumed[metric.ConsumerName]++
			insights.RpmConsumer.add(metric.TopicName, metric.ConsumerName, dateInMinute)
			continue

		} else {
			insights.TotalPublished++
			if metric.ACK {
				insights.TotalPublishedWithSuccess++
			} else {
				insights.TotalPublishedWithErr++
			}

			key := metric.TopicName
			df, ok := timeDurationsMsPublished[key]
			if ok {
				df = append(df, float64(metric.TimeDurationMs))
			} else {
				timeDurationsMsPublished[key] = []float64{float64(metric.TimeDurationMs)}
			}

			insights.TotalSegmentationPublished[metric.TopicName]++
			insights.RpmPublisher.add(metric.TopicName, dateInMinute)
		}

	}

	for key := range timeDurationsMsConsumed {
		p99, _ := percentile(timeDurationsMsConsumed[key], 0.99)
		p75, _ := percentile(timeDurationsMsConsumed[key], 0.75)
		insights.P99Consumed[key] = p99
		insights.P75Consumed[key] = p75
	}

	for key := range timeDurationsMsPublished {
		p99, _ := percentile(timeDurationsMsPublished[key], 0.99)
		p75, _ := percentile(timeDurationsMsPublished[key], 0.75)
		insights.P99Published[key] = p99
		insights.P75Published[key] = p75
	}

	insights.PercentageConsumedSuccess = fmt.Sprintf("%.2f%%", float64(insights.TotalConsumedWithSuccess)/float64(insights.TotalConsumed)*100)
	insights.PercentagePublishedSuccess = fmt.Sprintf("%.2f%%", float64(insights.TotalPublishedWithSuccess)/float64(insights.TotalPublished)*100)

	return insights
}

type Insights struct {
	// Generic infos
	TotalPublished             int64  `json:"total_published"`
	TotalConsumed              int64  `json:"total_consumed"`
	PercentagePublishedSuccess string `json:"percentage_published_success"`
	PercentageConsumedSuccess  string `json:"percentage_consumed_success"`
	TotalPublishedWithSuccess  int64  `json:"total_published_with_success"`
	TotalConsumedWithSuccess   int64  `json:"total_consumed_with_success"`
	TotalPublishedWithErr      int64  `json:"total_published_with_err"`
	TotalConsumedWithErr       int64  `json:"total_consumed_with_err"`
	//Segmentation infos
	TotalSegmentationPublished map[string]int64   `json:"total_segmentation_published"`
	TotalSegmentationConsumed  map[string]int64   `json:"total_segmentation_consumed"`
	RpmPublisher               RpmPublisher       `json:"rpm_publisher"`
	RpmConsumer                RpmConsumer        `json:"rpm_consumer"`
	P99Consumed                map[string]float64 `json:"consumers_p99"`  //Topic+Consumer >> value
	P75Consumed                map[string]float64 `json:"consumers_p75"`  //Topic+Consumer >> value
	P99Published               map[string]float64 `json:"publishers_p99"` //Topic >> value
	P75Published               map[string]float64 `json:"publishers_p75"` //Topic >> value
}

func NewInsights() Insights {
	return Insights{
		TotalPublished:             0,
		TotalConsumed:              0,
		TotalPublishedWithSuccess:  0,
		TotalConsumedWithSuccess:   0,
		TotalPublishedWithErr:      0,
		TotalConsumedWithErr:       0,
		PercentagePublishedSuccess: "",
		PercentageConsumedSuccess:  "",
		TotalSegmentationPublished: make(map[string]int64),
		TotalSegmentationConsumed:  make(map[string]int64),
		RpmPublisher:               make(map[string]*RPM), // [{time:"", value:2}]
		RpmConsumer:                make(map[string]*RPM),
		P99Consumed:                make(map[string]float64),
		P75Consumed:                make(map[string]float64),
		P99Published:               make(map[string]float64),
		P75Published:               make(map[string]float64),
	}
}

type RPM struct {
	Timeseries []time.Time `json:"timeseries"`
	Values     []int64     `json:"values"`
}

type RpmPublisher map[string]*RPM

func (rp RpmPublisher) add(topicName string, dateInMinute time.Time) {
	if data, ok := rp[topicName]; ok {
		lastDateTime := data.Timeseries[len(data.Timeseries)-1]
		if lastDateTime.Equal(dateInMinute) {
			data.Values[len(data.Values)-1]++
			return
		}
		data.Timeseries = append(data.Timeseries, dateInMinute)
		data.Values = append(data.Values, 1)
	} else {
		rp[topicName] = &RPM{Timeseries: []time.Time{dateInMinute}, Values: []int64{1}}
	}
}

type RpmConsumer map[string]*RPM

func (rc RpmConsumer) add(topicName, consumerName string, dateInMinute time.Time) {
	key := strings.Join([]string{topicName, consumerName}, ":")
	if data, ok := rc[key]; ok {
		lastDateTime := data.Timeseries[len(data.Timeseries)-1]
		if lastDateTime.Equal(dateInMinute) {
			data.Values[len(data.Values)-1]++
			return
		}
		data.Timeseries = append(data.Timeseries, dateInMinute)
		data.Values = append(data.Values, 1)
	} else {
		rc[key] = &RPM{Timeseries: []time.Time{dateInMinute}, Values: []int64{1}}
	}
}

func percentile(data []float64, p float64) (float64, error) {
	if len(data) == 0 {
		return 0, fmt.Errorf("empty array")
	}
	if p < 0 || p > 1 {
		return 0, fmt.Errorf("p must be between 0 and 1")
	}

	sorted := make([]float64, len(data))
	copy(sorted, data)
	sort.Float64s(sorted)

	n := float64(len(sorted))
	idx := int(math.Ceil(p*n) - 1)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}

	return sorted[idx], nil
}
