// parse_and_p99.go
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
)

// Evento representa o seu item JSON
type Evento struct {
	TopicName      string  `json:"TopicName"`
	ConsumerName   string  `json:"ConsumerName"`
	TimeStarted    string  `json:"TimeStarted"`
	TimeEnded      string  `json:"TimeEnded"`
	TimeDurationMs float64 `json:"TimeDurationMs"`
	ACK            bool    `json:"ACK"`
}

func percentile(data []float64, p float64) (float64, error) {
	if len(data) == 0 {
		return 0, fmt.Errorf("array vazio")
	}
	if p < 0 || p > 1 {
		return 0, fmt.Errorf("p deve estar entre 0 e 1")
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

func main() {
	// Coloque seu JSON como um array de objetos
	raw := `[
      {
        "TopicName": "payment.processed",
        "ConsumerName": "consumer-1",
        "TimeStarted": "2025-10-11T14:26:49.66209-03:00",
        "TimeEnded": "2025-10-11T14:26:49.663211-03:00",
        "TimeDurationMs": 1,
        "ACK": true
      },
      {
        "TopicName": "payment.processed",
        "ConsumerName": "consumer-1",
        "TimeStarted": "2025-10-11T14:26:50.450483-03:00",
        "TimeEnded": "2025-10-11T14:26:50.451332-03:00",
        "TimeDurationMs": 0,
        "ACK": true
      },
      {
        "TopicName": "payment.processed",
        "ConsumerName": "consumer-1",
        "TimeStarted": "2025-10-11T14:26:50.896034-03:00",
        "TimeEnded": "2025-10-11T14:26:50.896797-03:00",
        "TimeDurationMs": 0,
        "ACK": true
      },
      {
        "TopicName": "payment.processed",
        "ConsumerName": "consumer-1",
        "TimeStarted": "2025-10-11T14:26:51.406911-03:00",
        "TimeEnded": "2025-10-11T14:26:51.407643-03:00",
        "TimeDurationMs": 0,
        "ACK": true
      }
    ]`

	var eventos []Evento
	if err := json.Unmarshal([]byte(raw), &eventos); err != nil {
		panic(err)
	}

	// Extrai TimeDurationMs
	durations := make([]float64, 0, len(eventos))
	for _, e := range eventos {
		durations = append(durations, e.TimeDurationMs)
	}

	p99, err := percentile(durations, 0.99)
	if err != nil {
		panic(err)
	}
	fmt.Printf("p99: %.0f ms\n", p99) // esperado: 1 ms
}
