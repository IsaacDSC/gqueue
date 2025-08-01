package interweb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IsaacDSC/webhook/internal/consworker"
	"github.com/IsaacDSC/webhook/pkg/publisher"
	"go.uber.org/mock/gomock"
)

func TestHandler(t *testing.T) {
	tests := []struct {
		name           string
		payload        interface{}
		setupMock      func(*testing.T, *publisher.MockPublisher)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "successful request with valid payload",
			payload: Payload{
				EventName: "user.created",
				Data: Data{
					"user_id": "123",
					"email":   "test@example.com",
				},
			},
			setupMock: func(t *testing.T, mockPub *publisher.MockPublisher) {
				// Use custom matcher to validate the payload structure
				mockPub.EXPECT().
					Publish(gomock.Any(), consworker.PublisherExternalEvent.String(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, eventName string, payload Payload, opts ...interface{}) error {
						if payload.EventName != "user.created" {
							t.Errorf("Expected event name 'user.created', got '%s'", payload.EventName)
						}
						if payload.Data["user_id"] != "123" {
							t.Errorf("Expected user_id '123', got '%v'", payload.Data["user_id"])
						}
						if payload.Data["email"] != "test@example.com" {
							t.Errorf("Expected email 'test@example.com', got '%v'", payload.Data["email"])
						}
						return nil
					})
			},
			expectedStatus: http.StatusAccepted,
			expectedBody:   "",
		},
		{
			name: "successful request with empty data",
			payload: Payload{
				EventName: "ping",
				Data:      Data{},
			},
			setupMock: func(t *testing.T, mockPub *publisher.MockPublisher) {
				mockPub.EXPECT().
					Publish(gomock.Any(), consworker.PublisherExternalEvent.String(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, eventName string, payload Payload, opts ...interface{}) error {
						if payload.EventName != "ping" {
							t.Errorf("Expected event name 'ping', got '%s'", payload.EventName)
						}
						if len(payload.Data) != 0 {
							t.Errorf("Expected empty data, got %v", payload.Data)
						}
						return nil
					})
			},
			expectedStatus: http.StatusAccepted,
			expectedBody:   "",
		},
		{
			name: "successful request with complex data",
			payload: Payload{
				EventName: "order.completed",
				Data: Data{
					"order_id": "order-123",
					"customer": map[string]interface{}{
						"id":   "customer-456",
						"name": "John Doe",
					},
					"items": []interface{}{
						map[string]interface{}{
							"id":       "item-1",
							"quantity": 2,
							"price":    29.99,
						},
						map[string]interface{}{
							"id":       "item-2",
							"quantity": 1,
							"price":    15.50,
						},
					},
					"total": 75.48,
				},
			},
			setupMock: func(t *testing.T, mockPub *publisher.MockPublisher) {
				// Use gomock.Any() for payload since JSON unmarshaling can change float representation
				mockPub.EXPECT().
					Publish(gomock.Any(), consworker.PublisherExternalEvent.String(), gomock.Any()).
					Return(nil)
			},
			expectedStatus: http.StatusAccepted,
			expectedBody:   "",
		},
		{
			name:           "invalid JSON payload",
			payload:        `{"event_name": "invalid", "data": }`,
			setupMock:      func(t *testing.T, mockPub *publisher.MockPublisher) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid character '}' looking for beginning of value\n",
		},
		{
			name: "missing event_name field",
			payload: map[string]interface{}{
				"data": map[string]interface{}{
					"key": "value",
				},
			},
			setupMock: func(t *testing.T, mockPub *publisher.MockPublisher) {
				mockPub.EXPECT().
					Publish(gomock.Any(), consworker.PublisherExternalEvent.String(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, eventName string, payload Payload, opts ...interface{}) error {
						if payload.EventName != "" {
							t.Errorf("Expected empty event name, got '%s'", payload.EventName)
						}
						if payload.Data["key"] != "value" {
							t.Errorf("Expected key 'value', got '%v'", payload.Data["key"])
						}
						return nil
					})
			},
			expectedStatus: http.StatusAccepted,
			expectedBody:   "",
		},
		{
			name: "missing data field",
			payload: map[string]interface{}{
				"event_name": "test.event",
			},
			setupMock: func(t *testing.T, mockPub *publisher.MockPublisher) {
				mockPub.EXPECT().
					Publish(gomock.Any(), consworker.PublisherExternalEvent.String(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, eventName string, payload Payload, opts ...interface{}) error {
						if payload.EventName != "test.event" {
							t.Errorf("Expected event name 'test.event', got '%s'", payload.EventName)
						}
						if payload.Data != nil {
							t.Errorf("Expected nil data, got %v", payload.Data)
						}
						return nil
					})
			},
			expectedStatus: http.StatusAccepted,
			expectedBody:   "",
		},
		{
			name: "publisher returns error",
			payload: Payload{
				EventName: "failing.event",
				Data: Data{
					"test": "data",
				},
			},
			setupMock: func(t *testing.T, mockPub *publisher.MockPublisher) {
				mockPub.EXPECT().
					Publish(gomock.Any(), consworker.PublisherExternalEvent.String(), gomock.Any()).
					Return(errors.New("publisher failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "publisher failed\n",
		},
		{
			name: "publisher returns context error",
			payload: Payload{
				EventName: "context.error",
				Data: Data{
					"test": "data",
				},
			},
			setupMock: func(t *testing.T, mockPub *publisher.MockPublisher) {
				mockPub.EXPECT().
					Publish(gomock.Any(), consworker.PublisherExternalEvent.String(), gomock.Any()).
					Return(context.DeadlineExceeded)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "context deadline exceeded\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockPublisher := publisher.NewMockPublisher(ctrl)
			tt.setupMock(t, mockPublisher)

			handler := GetPublisherHandle(mockPublisher)

			// Prepare request body
			var reqBody []byte
			var err error
			if str, ok := tt.payload.(string); ok {
				reqBody = []byte(str)
			} else {
				reqBody, err = json.Marshal(tt.payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
			}

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/internal/publish", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Execute
			handler(w, req)

			// Assert
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody != "" && w.Body.String() != tt.expectedBody {
				t.Errorf("Expected body %q, got %q", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestData_ToBytes(t *testing.T) {
	tests := []struct {
		name     string
		data     Data
		expected string
	}{
		{
			name:     "empty data",
			data:     Data{},
			expected: "{}",
		},
		{
			name: "simple data",
			data: Data{
				"key1": "value1",
				"key2": 123,
			},
			expected: `{"key1":"value1","key2":123}`,
		},
		{
			name: "nested data",
			data: Data{
				"user": map[string]interface{}{
					"id":   "123",
					"name": "John",
				},
				"active": true,
			},
			expected: `{"active":true,"user":{"id":"123","name":"John"}}`,
		},
		{
			name: "data with array",
			data: Data{
				"items": []interface{}{"item1", "item2", "item3"},
				"count": 3,
			},
			expected: `{"count":3,"items":["item1","item2","item3"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.data.ToBytes()

			// Since JSON marshal doesn't guarantee order, we need to compare the actual JSON content
			var expectedJSON, resultJSON interface{}
			if err := json.Unmarshal([]byte(tt.expected), &expectedJSON); err != nil {
				t.Fatalf("Failed to unmarshal expected JSON: %v", err)
			}
			if err := json.Unmarshal(result, &resultJSON); err != nil {
				t.Fatalf("Failed to unmarshal result JSON: %v", err)
			}

			expectedBytes, _ := json.Marshal(expectedJSON)
			resultBytes, _ := json.Marshal(resultJSON)

			if string(expectedBytes) != string(resultBytes) {
				t.Errorf("Expected %s, got %s", string(expectedBytes), string(resultBytes))
			}
		})
	}
}

func TestPayload_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name    string
		payload Payload
	}{
		{
			name: "simple payload",
			payload: Payload{
				EventName: "test.event",
				Data: Data{
					"key": "value",
				},
			},
		},
		{
			name: "complex payload",
			payload: Payload{
				EventName: "complex.event",
				Data: Data{
					"user": map[string]interface{}{
						"id":    "123",
						"email": "test@example.com",
						"preferences": map[string]interface{}{
							"theme":         "dark",
							"notifications": true,
						},
					},
					"timestamp": "2023-01-01T00:00:00Z",
					"metadata": []interface{}{
						"tag1", "tag2", "tag3",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			jsonBytes, err := json.Marshal(tt.payload)
			if err != nil {
				t.Fatalf("Failed to marshal payload: %v", err)
			}

			// Unmarshal back to Payload
			var unmarshaled Payload
			if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal payload: %v", err)
			}

			// Compare event names
			if unmarshaled.EventName != tt.payload.EventName {
				t.Errorf("Expected event name %s, got %s", tt.payload.EventName, unmarshaled.EventName)
			}

			// Compare data by marshaling both to JSON and comparing
			originalDataBytes, _ := json.Marshal(tt.payload.Data)
			unmarshaledDataBytes, _ := json.Marshal(unmarshaled.Data)

			if string(originalDataBytes) != string(unmarshaledDataBytes) {
				t.Errorf("Data mismatch. Original: %s, Unmarshaled: %s",
					string(originalDataBytes), string(unmarshaledDataBytes))
			}
		})
	}
}
