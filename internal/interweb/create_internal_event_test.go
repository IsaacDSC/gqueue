package interweb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/IsaacDSC/webhook/internal/intersvc"
	"github.com/google/uuid"
)

func TestGetCreateHandle(t *testing.T) {
	tests := []struct {
		name                 string
		requestBody          interface{}
		createEventFunc      CreateEventFunc
		expectedStatus       int
		expectedBodyContains string
		expectError          bool
	}{
		{
			name: "successful creation with valid payload",
			requestBody: intersvc.CreateInternalEventDto{
				EventName:   "user.created",
				ServiceName: "user-service",
				RepoUrl:     "https://github.com/example/user-service",
				TeamOwner:   "backend-team",
			},
			createEventFunc: func(ctx context.Context, input intersvc.CreateInternalEventDto) (intersvc.InternalEvent, error) {
				return intersvc.InternalEvent{
					ID:          uuid.New(),
					Name:        input.EventName,
					ServiceName: input.ServiceName,
					RepoUrl:     input.RepoUrl,
					TeamOwner:   input.TeamOwner,
					Triggers:    []intersvc.Trigger{},
				}, nil
			},
			expectedStatus:       http.StatusCreated,
			expectedBodyContains: "user.created",
			expectError:          false,
		},
		{
			name:        "invalid JSON payload",
			requestBody: `{"invalid": json}`,
			createEventFunc: func(ctx context.Context, input intersvc.CreateInternalEventDto) (intersvc.InternalEvent, error) {
				return intersvc.InternalEvent{}, nil
			},
			expectedStatus:       http.StatusBadRequest,
			expectedBodyContains: "Invalid request body",
			expectError:          true,
		},
		{
			name: "create event function returns error",
			requestBody: intersvc.CreateInternalEventDto{
				EventName:   "user.created",
				ServiceName: "user-service",
				RepoUrl:     "https://github.com/example/user-service",
				TeamOwner:   "backend-team",
			},
			createEventFunc: func(ctx context.Context, input intersvc.CreateInternalEventDto) (intersvc.InternalEvent, error) {
				return intersvc.InternalEvent{}, errors.New("internal event already exists")
			},
			expectedStatus:       http.StatusInternalServerError,
			expectedBodyContains: "Failed to create internal event",
			expectError:          true,
		},
		{
			name: "empty request body",
			requestBody: intersvc.CreateInternalEventDto{
				EventName:   "",
				ServiceName: "",
				RepoUrl:     "",
				TeamOwner:   "",
			},
			createEventFunc: func(ctx context.Context, input intersvc.CreateInternalEventDto) (intersvc.InternalEvent, error) {
				return intersvc.InternalEvent{
					ID:          uuid.New(),
					Name:        input.EventName,
					ServiceName: input.ServiceName,
					RepoUrl:     input.RepoUrl,
					TeamOwner:   input.TeamOwner,
					Triggers:    []intersvc.Trigger{},
				}, nil
			},
			expectedStatus:       http.StatusCreated,
			expectedBodyContains: "",
			expectError:          false,
		},
		{
			name: "partial data in request body",
			requestBody: intersvc.CreateInternalEventDto{
				EventName:   "order.created",
				ServiceName: "order-service",
				RepoUrl:     "",
				TeamOwner:   "",
			},
			createEventFunc: func(ctx context.Context, input intersvc.CreateInternalEventDto) (intersvc.InternalEvent, error) {
				return intersvc.InternalEvent{
					ID:          uuid.New(),
					Name:        input.EventName,
					ServiceName: input.ServiceName,
					RepoUrl:     input.RepoUrl,
					TeamOwner:   input.TeamOwner,
					Triggers:    []intersvc.Trigger{},
				}, nil
			},
			expectedStatus:       http.StatusCreated,
			expectedBodyContains: "order.created",
			expectError:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare request body
			var requestBody []byte
			var err error

			if str, ok := tt.requestBody.(string); ok {
				// Handle invalid JSON string
				requestBody = []byte(str)
			} else {
				// Handle valid struct
				requestBody, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			// Create HTTP request
			req, err := http.NewRequest(http.MethodPost, "/internal-events", bytes.NewBuffer(requestBody))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			rr := httptest.NewRecorder()

			// Get the handler
			handler := GetCreateEventHandle(tt.createEventFunc)

			// Execute the handler
			handler(rr, req)

			// Verify status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Verify response body contains expected content
			if tt.expectedBodyContains != "" {
				responseBody := rr.Body.String()
				if !strings.Contains(responseBody, tt.expectedBodyContains) {
					t.Errorf("Expected response body to contain '%s', got '%s'", tt.expectedBodyContains, responseBody)
				}
			}

			// Verify response content for successful cases
			if !tt.expectError && rr.Code == http.StatusCreated {
				var response intersvc.InternalEvent
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
				}

				// Verify the response contains expected data
				if dto, ok := tt.requestBody.(intersvc.CreateInternalEventDto); ok {
					if response.Name != dto.EventName {
						t.Errorf("Expected response name '%s', got '%s'", dto.EventName, response.Name)
					}
					if response.ServiceName != dto.ServiceName {
						t.Errorf("Expected response service name '%s', got '%s'", dto.ServiceName, response.ServiceName)
					}
				}
			}
		})
	}
}

func TestGetCreateHandle_ContextPropagation(t *testing.T) {
	contextReceived := false

	createEventFunc := func(ctx context.Context, input intersvc.CreateInternalEventDto) (intersvc.InternalEvent, error) {
		if ctx != nil {
			contextReceived = true
		}
		return intersvc.InternalEvent{
			ID:          uuid.New(),
			Name:        input.EventName,
			ServiceName: input.ServiceName,
			RepoUrl:     input.RepoUrl,
			TeamOwner:   input.TeamOwner,
			Triggers:    []intersvc.Trigger{},
		}, nil
	}

	requestBody := intersvc.CreateInternalEventDto{
		EventName:   "test.event",
		ServiceName: "test-service",
		RepoUrl:     "https://github.com/test/repo",
		TeamOwner:   "test-team",
	}

	body, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest(http.MethodPost, "/internal-events", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := GetCreateEventHandle(createEventFunc)
	handler(rr, req)

	if !contextReceived {
		t.Error("Expected context to be passed to createEventFunc")
	}

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, rr.Code)
	}
}

func TestGetCreateHandle_JSONEncodingError(t *testing.T) {
	// Create a mock that returns a struct that cannot be JSON marshaled
	createEventFunc := func(ctx context.Context, input intersvc.CreateInternalEventDto) (intersvc.InternalEvent, error) {
		// Return a response that would cause JSON encoding to fail
		// Note: In this case, we simulate this by having the handler fail during encoding
		return intersvc.InternalEvent{
			ID:          uuid.New(),
			Name:        input.EventName,
			ServiceName: input.ServiceName,
			RepoUrl:     input.RepoUrl,
			TeamOwner:   input.TeamOwner,
			Triggers:    []intersvc.Trigger{},
		}, nil
	}

	requestBody := intersvc.CreateInternalEventDto{
		EventName:   "test.event",
		ServiceName: "test-service",
		RepoUrl:     "https://github.com/test/repo",
		TeamOwner:   "test-team",
	}

	body, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest(http.MethodPost, "/internal-events", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Create a ResponseWriter that will fail on Write
	rr := &FailingResponseWriter{
		ResponseRecorder: httptest.NewRecorder(),
		failOnWrite:      true,
	}

	handler := GetCreateEventHandle(createEventFunc)
	handler(rr, req)

	// Since we can't easily test JSON encoding failure without modifying the handler,
	// this test demonstrates the structure. In a real scenario, you might use dependency injection
	// to make the JSON encoder mockable.
}

// FailingResponseWriter is a test helper that can simulate write failures
type FailingResponseWriter struct {
	*httptest.ResponseRecorder
	failOnWrite bool
}

func (f *FailingResponseWriter) Write(b []byte) (int, error) {
	if f.failOnWrite {
		return 0, errors.New("simulated write failure")
	}
	return f.ResponseRecorder.Write(b)
}

// Benchmark test for the handler
func BenchmarkGetCreateHandle(b *testing.B) {
	createEventFunc := func(ctx context.Context, input intersvc.CreateInternalEventDto) (intersvc.InternalEvent, error) {
		return intersvc.InternalEvent{
			ID:          uuid.New(),
			Name:        input.EventName,
			ServiceName: input.ServiceName,
			RepoUrl:     input.RepoUrl,
			TeamOwner:   input.TeamOwner,
			Triggers:    []intersvc.Trigger{},
		}, nil
	}

	requestBody := intersvc.CreateInternalEventDto{
		EventName:   "bench.event",
		ServiceName: "bench-service",
		RepoUrl:     "https://github.com/bench/repo",
		TeamOwner:   "bench-team",
	}

	body, _ := json.Marshal(requestBody)
	handler := GetCreateEventHandle(createEventFunc)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest(http.MethodPost, "/internal-events", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler(rr, req)
	}
}
