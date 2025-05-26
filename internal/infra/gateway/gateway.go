package gateway

import (
	"bytes"
	"context"
	"fmt"
	"github.com/IsaacDSC/webhook/internal/structs"
	"net/http"
)

type Hook struct {
}

func NewHook() *Hook {
	return &Hook{}
}

func (h Hook) Publisher(ctx context.Context, event structs.ExternalEvent) (sent []structs.Trigger, err error) {
	if len(event.Triggers) == 0 {
		return
	}

	var filteredTriggers []structs.Trigger
	if len(event.Delivered) > 0 {
		// Create a map for O(1) lookup of delivered triggers
		deliveredMap := make(map[string]bool)
		for _, delivered := range event.Delivered {
			// Create a unique key based on BaseUrl and Path
			key := fmt.Sprintf("%s/%s", delivered.BaseUrl, delivered.Path)
			deliveredMap[key] = true
		}

		// Filter out triggers that are already in the delivered list
		for _, trigger := range event.Triggers {
			key := fmt.Sprintf("%s/%s", trigger.BaseUrl, trigger.Path)
			if !deliveredMap[key] {
				filteredTriggers = append(filteredTriggers, trigger)
			}
		}
	} else {
		// If no delivered triggers, use all triggers
		filteredTriggers = event.Triggers
	}

	var successTriggers []structs.Trigger
	for _, trigger := range filteredTriggers {
		url := fmt.Sprintf("%s/%s", trigger.BaseUrl, trigger.Path)
		bodyReader := bytes.NewReader(event.Data.ToBytes())
		resp, err := http.Post(url, "application/json", bodyReader)
		if err != nil {
			continue
		}

		if resp.StatusCode != http.StatusOK {
			continue
		}

		successTriggers = append(successTriggers, trigger)
	}

	return successTriggers, nil
}
