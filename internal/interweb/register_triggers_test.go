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
	"time"

	"github.com/IsaacDSC/webhook/internal/intersvc"
	"github.com/google/uuid"
)

func TestGetRegisterHandle(t *testing.T) {
	tests := []struct {
		name                 string
		requestBody          interface{}
		registerTriggerFunc  RegisterTriggerFunc
		expectedStatus       int
		expectedBodyContains string
		expectError          bool
	}{
		{
			name: "successful trigger registration with valid payload",
			requestBody: intersvc.RegisterTriggersDto{
				EventName: "user.created",
				Trigger: intersvc.TriggerDto{
					ServiceName: "notification-service",
					Type:        "persistent",
					BaseUrl:     "https://api.notification.com",
					Path:        "/webhooks/user-created",
				},
			},
			registerTriggerFunc: func(ctx context.Context, input intersvc.RegisterTriggersDto) (intersvc.InternalEvent, error) {
				trigger := input.ToTrigger()
				return intersvc.InternalEvent{
					ID:          uuid.New(),
					Name:        input.EventName,
					ServiceName: "user-service",
					RepoUrl:     "https://github.com/example/user-service",
					TeamOwner:   "backend-team",
					Triggers:    intersvc.ListTrigger{trigger},
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}, nil
			},
			expectedStatus:       http.StatusCreated,
			expectedBodyContains: "user.created",
			expectError:          false,
		},
		{
			name:        "invalid JSON payload",
			requestBody: `{"invalid": json}`,
			registerTriggerFunc: func(ctx context.Context, input intersvc.RegisterTriggersDto) (intersvc.InternalEvent, error) {
				return intersvc.InternalEvent{}, nil
			},
			expectedStatus:       http.StatusBadRequest,
			expectedBodyContains: "Invalid request body",
			expectError:          true,
		},
		{
			name: "register trigger function returns error",
			requestBody: intersvc.RegisterTriggersDto{
				EventName: "user.created",
				Trigger: intersvc.TriggerDto{
					ServiceName: "notification-service",
					Type:        "persistent",
					BaseUrl:     "https://api.notification.com",
					Path:        "/webhooks/user-created",
				},
			},
			registerTriggerFunc: func(ctx context.Context, input intersvc.RegisterTriggersDto) (intersvc.InternalEvent, error) {
				return intersvc.InternalEvent{}, errors.New("event not found")
			},
			expectedStatus:       http.StatusInternalServerError,
			expectedBodyContains: "Failed to register trigger",
			expectError:          true,
		},
		{
			name: "registration with different trigger types",
			requestBody: intersvc.RegisterTriggersDto{
				EventName: "order.created",
				Trigger: intersvc.TriggerDto{
					ServiceName: "inventory-service",
					Type:        "fireForGet",
					BaseUrl:     "https://api.inventory.com",
					Path:        "/webhooks/order-created",
				},
			},
			registerTriggerFunc: func(ctx context.Context, input intersvc.RegisterTriggersDto) (intersvc.InternalEvent, error) {
				trigger := input.ToTrigger()
				return intersvc.InternalEvent{
					ID:          uuid.New(),
					Name:        input.EventName,
					ServiceName: "order-service",
					RepoUrl:     "https://github.com/example/order-service",
					TeamOwner:   "backend-team",
					Triggers:    intersvc.ListTrigger{trigger},
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}, nil
			},
			expectedStatus:       http.StatusCreated,
			expectedBodyContains: "order.created",
			expectError:          false,
		},
		{
			name: "registration with notPersistent trigger type",
			requestBody: intersvc.RegisterTriggersDto{
				EventName: "payment.processed",
				Trigger: intersvc.TriggerDto{
					ServiceName: "email-service",
					Type:        "notPersistent",
					BaseUrl:     "https://api.email.com",
					Path:        "/webhooks/payment-processed",
				},
			},
			registerTriggerFunc: func(ctx context.Context, input intersvc.RegisterTriggersDto) (intersvc.InternalEvent, error) {
				trigger := input.ToTrigger()
				return intersvc.InternalEvent{
					ID:          uuid.New(),
					Name:        input.EventName,
					ServiceName: "payment-service",
					RepoUrl:     "https://github.com/example/payment-service",
					TeamOwner:   "payments-team",
					Triggers:    intersvc.ListTrigger{trigger},
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}, nil
			},
			expectedStatus:       http.StatusCreated,
			expectedBodyContains: "payment.processed",
			expectError:          false,
		},
		{
			name: "empty event name",
			requestBody: intersvc.RegisterTriggersDto{
				EventName: "",
				Trigger: intersvc.TriggerDto{
					ServiceName: "test-service",
					Type:        "persistent",
					BaseUrl:     "https://api.test.com",
					Path:        "/webhooks/test",
				},
			},
			registerTriggerFunc: func(ctx context.Context, input intersvc.RegisterTriggersDto) (intersvc.InternalEvent, error) {
				trigger := input.ToTrigger()
				return intersvc.InternalEvent{
					ID:          uuid.New(),
					Name:        input.EventName,
					ServiceName: "test-service",
					RepoUrl:     "https://github.com/example/test-service",
					TeamOwner:   "test-team",
					Triggers:    intersvc.ListTrigger{trigger},
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}, nil
			},
			expectedStatus:       http.StatusCreated,
			expectedBodyContains: "",
			expectError:          false,
		},
		{
			name: "empty trigger fields",
			requestBody: intersvc.RegisterTriggersDto{
				EventName: "test.event",
				Trigger: intersvc.TriggerDto{
					ServiceName: "",
					Type:        "",
					BaseUrl:     "",
					Path:        "",
				},
			},
			registerTriggerFunc: func(ctx context.Context, input intersvc.RegisterTriggersDto) (intersvc.InternalEvent, error) {
				trigger := input.ToTrigger()
				return intersvc.InternalEvent{
					ID:          uuid.New(),
					Name:        input.EventName,
					ServiceName: "test-service",
					RepoUrl:     "https://github.com/example/test-service",
					TeamOwner:   "test-team",
					Triggers:    intersvc.ListTrigger{trigger},
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}, nil
			},
			expectedStatus:       http.StatusCreated,
			expectedBodyContains: "test.event",
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
			req, err := http.NewRequest(http.MethodPost, "/register-triggers", bytes.NewBuffer(requestBody))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			rr := httptest.NewRecorder()

			// Get the handler
			handler := GetRegisterHandle(tt.registerTriggerFunc)

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
				if dto, ok := tt.requestBody.(intersvc.RegisterTriggersDto); ok {
					if response.Name != dto.EventName {
						t.Errorf("Expected response name '%s', got '%s'", dto.EventName, response.Name)
					}
					if len(response.Triggers) == 0 {
						t.Error("Expected response to contain triggers, got empty list")
					}
				}
			}
		})
	}
}

func TestGetRegisterHandle_ContextPropagation(t *testing.T) {
	contextReceived := false

	registerTriggerFunc := func(ctx context.Context, input intersvc.RegisterTriggersDto) (intersvc.InternalEvent, error) {
		if ctx != nil {
			contextReceived = true
		}
		trigger := input.ToTrigger()
		return intersvc.InternalEvent{
			ID:          uuid.New(),
			Name:        input.EventName,
			ServiceName: "test-service",
			RepoUrl:     "https://github.com/test/repo",
			TeamOwner:   "test-team",
			Triggers:    intersvc.ListTrigger{trigger},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}, nil
	}

	requestBody := intersvc.RegisterTriggersDto{
		EventName: "test.event",
		Trigger: intersvc.TriggerDto{
			ServiceName: "test-service",
			Type:        "persistent",
			BaseUrl:     "https://api.test.com",
			Path:        "/webhooks/test",
		},
	}

	body, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest(http.MethodPost, "/register-triggers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := GetRegisterHandle(registerTriggerFunc)
	handler(rr, req)

	if !contextReceived {
		t.Error("Expected context to be passed to registerTriggerFunc")
	}

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, rr.Code)
	}
}

func TestGetRegisterHandle_JSONEncodingError(t *testing.T) {
	registerTriggerFunc := func(ctx context.Context, input intersvc.RegisterTriggersDto) (intersvc.InternalEvent, error) {
		trigger := input.ToTrigger()
		return intersvc.InternalEvent{
			ID:          uuid.New(),
			Name:        input.EventName,
			ServiceName: "test-service",
			RepoUrl:     "https://github.com/test/repo",
			TeamOwner:   "test-team",
			Triggers:    intersvc.ListTrigger{trigger},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}, nil
	}

	requestBody := intersvc.RegisterTriggersDto{
		EventName: "test.event",
		Trigger: intersvc.TriggerDto{
			ServiceName: "test-service",
			Type:        "persistent",
			BaseUrl:     "https://api.test.com",
			Path:        "/webhooks/test",
		},
	}

	body, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest(http.MethodPost, "/register-triggers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Create a ResponseWriter that will fail on Write
	rr := &FailingResponseWriter{
		ResponseRecorder: httptest.NewRecorder(),
		failOnWrite:      true,
	}

	handler := GetRegisterHandle(registerTriggerFunc)
	handler(rr, req)

	// This test demonstrates the structure for testing JSON encoding failures
	// In practice, the exact behavior would depend on the specific implementation
}

func TestGetRegisterHandle_TriggerConversion(t *testing.T) {
	var receivedInput intersvc.RegisterTriggersDto

	registerTriggerFunc := func(ctx context.Context, input intersvc.RegisterTriggersDto) (intersvc.InternalEvent, error) {
		receivedInput = input
		trigger := input.ToTrigger()

		// Verify the trigger conversion works correctly
		if trigger.ServiceName != input.Trigger.ServiceName {
			t.Error("Trigger conversion failed: ServiceName mismatch")
		}
		if trigger.BaseUrl != input.Trigger.BaseUrl {
			t.Error("Trigger conversion failed: BaseUrl mismatch")
		}
		if trigger.Path != input.Trigger.Path {
			t.Error("Trigger conversion failed: Path mismatch")
		}
		if string(trigger.Type) != input.Trigger.Type {
			t.Error("Trigger conversion failed: Type mismatch")
		}

		return intersvc.InternalEvent{
			ID:          uuid.New(),
			Name:        input.EventName,
			ServiceName: "test-service",
			RepoUrl:     "https://github.com/test/repo",
			TeamOwner:   "test-team",
			Triggers:    intersvc.ListTrigger{trigger},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}, nil
	}

	requestBody := intersvc.RegisterTriggersDto{
		EventName: "conversion.test",
		Trigger: intersvc.TriggerDto{
			ServiceName: "conversion-service",
			Type:        "fireForGet",
			BaseUrl:     "https://api.conversion.com",
			Path:        "/webhooks/conversion-test",
		},
	}

	body, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest(http.MethodPost, "/register-triggers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := GetRegisterHandle(registerTriggerFunc)
	handler(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, rr.Code)
	}

	// Verify the input was received correctly
	if receivedInput.EventName != requestBody.EventName {
		t.Errorf("Expected EventName '%s', got '%s'", requestBody.EventName, receivedInput.EventName)
	}
	if receivedInput.Trigger.ServiceName != requestBody.Trigger.ServiceName {
		t.Errorf("Expected ServiceName '%s', got '%s'", requestBody.Trigger.ServiceName, receivedInput.Trigger.ServiceName)
	}
}

// Benchmark test for the handler
func BenchmarkGetRegisterHandle(b *testing.B) {
	registerTriggerFunc := func(ctx context.Context, input intersvc.RegisterTriggersDto) (intersvc.InternalEvent, error) {
		trigger := input.ToTrigger()
		return intersvc.InternalEvent{
			ID:          uuid.New(),
			Name:        input.EventName,
			ServiceName: "bench-service",
			RepoUrl:     "https://github.com/bench/repo",
			TeamOwner:   "bench-team",
			Triggers:    intersvc.ListTrigger{trigger},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}, nil
	}

	requestBody := intersvc.RegisterTriggersDto{
		EventName: "bench.event",
		Trigger: intersvc.TriggerDto{
			ServiceName: "bench-service",
			Type:        "persistent",
			BaseUrl:     "https://api.bench.com",
			Path:        "/webhooks/bench",
		},
	}

	body, _ := json.Marshal(requestBody)
	handler := GetRegisterHandle(registerTriggerFunc)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest(http.MethodPost, "/register-triggers", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler(rr, req)
	}
}
