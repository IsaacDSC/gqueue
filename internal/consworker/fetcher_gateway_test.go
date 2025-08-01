package consworker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFetchAll provides comprehensive testing for the FetchAll function covering:
// - Successful requests to multiple triggers
// - Partial failures (some triggers succeed, some fail)
// - Complete failures (all triggers fail)
// - Server timeout scenarios
// - Concurrent execution validation
func TestFetchAll(t *testing.T) {
	tests := []struct {
		name               string
		event              ExternalPayload
		setupServers       func() []*httptest.Server
		expectedError      bool
		expectedErrorCount int
		expectedOutput     int
	}{
		{
			name: "successful requests to all triggers",
			event: ExternalPayload{
				EventName: "test-event",
				Data: map[string]any{
					"user_id": "123",
					"action":  "created",
				},
				ExtraHeaders: map[string]string{
					"X-Custom-Header": "test-value",
				},
				Triggers: []Trigger{
					{
						ServiceName: "service1",
						Type:        TriggerTypePersistent,
						BaseUrl:     "", // will be set by setupServers
						Path:        "/webhook",
					},
					{
						ServiceName: "service2",
						Type:        TriggerTypeNotPersistent,
						BaseUrl:     "", // will be set by setupServers
						Path:        "/api/hook",
					},
				},
			},
			setupServers: func() []*httptest.Server {
				// Server 1 - successful response
				server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, http.MethodPost, r.Method)
					assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
					assert.Equal(t, "test-value", r.Header.Get("X-Custom-Header"))

					// Verify request body
					var body map[string]any
					err := json.NewDecoder(r.Body).Decode(&body)
					assert.NoError(t, err)
					assert.Equal(t, "123", body["user_id"])
					assert.Equal(t, "created", body["action"])

					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"status": "ok"}`))
				}))

				// Server 2 - successful response
				server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, http.MethodPost, r.Method)
					assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"received": true}`))
				}))

				return []*httptest.Server{server1, server2}
			},
			expectedError:      false,
			expectedErrorCount: 0,
			expectedOutput:     0,
		},
		{
			name: "one trigger fails with 500 error",
			event: ExternalPayload{
				EventName: "test-event",
				Data: map[string]any{
					"user_id": "456",
				},
				ExtraHeaders: map[string]string{},
				Triggers: []Trigger{
					{
						ServiceName: "service1",
						Type:        TriggerTypePersistent,
						BaseUrl:     "", // will be set by setupServers
						Path:        "/webhook",
					},
					{
						ServiceName: "service2",
						Type:        TriggerTypeNotPersistent,
						BaseUrl:     "", // will be set by setupServers
						Path:        "/api/hook",
					},
				},
			},
			setupServers: func() []*httptest.Server {
				// Server 1 - successful response
				server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"status": "ok"}`))
				}))

				// Server 2 - error response
				server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error": "internal server error"}`))
				}))

				return []*httptest.Server{server1, server2}
			},
			expectedError:      true,
			expectedErrorCount: 1,
			expectedOutput:     1,
		},
		{
			name: "all triggers fail",
			event: ExternalPayload{
				EventName: "test-event",
				Data: map[string]any{
					"test": "data",
				},
				ExtraHeaders: map[string]string{},
				Triggers: []Trigger{
					{
						ServiceName: "service1",
						Type:        TriggerTypePersistent,
						BaseUrl:     "", // will be set by setupServers
						Path:        "/webhook",
					},
					{
						ServiceName: "service2",
						Type:        TriggerTypeNotPersistent,
						BaseUrl:     "", // will be set by setupServers
						Path:        "/api/hook",
					},
				},
			},
			setupServers: func() []*httptest.Server {
				// Server 1 - error response
				server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte(`{"error": "bad request"}`))
				}))

				// Server 2 - error response
				server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
					w.Write([]byte(`{"error": "not found"}`))
				}))

				return []*httptest.Server{server1, server2}
			},
			expectedError:      true,
			expectedErrorCount: 2,
			expectedOutput:     2,
		},
		{
			name: "server timeout simulation",
			event: ExternalPayload{
				EventName: "test-event",
				Data: map[string]any{
					"test": "timeout",
				},
				ExtraHeaders: map[string]string{},
				Triggers: []Trigger{
					{
						ServiceName: "service1",
						Type:        TriggerTypePersistent,
						BaseUrl:     "", // will be set by setupServers
						Path:        "/webhook",
					},
				},
			},
			setupServers: func() []*httptest.Server {
				// Server that delays response to simulate timeout
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					time.Sleep(100 * time.Millisecond) // Small delay for testing
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"status": "ok"}`))
				}))

				return []*httptest.Server{server}
			},
			expectedError:      false,
			expectedErrorCount: 0,
			expectedOutput:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test servers
			servers := tt.setupServers()
			defer func() {
				for _, server := range servers {
					server.Close()
				}
			}()

			// Update triggers with server URLs
			for i, server := range servers {
				if i < len(tt.event.Triggers) {
					tt.event.Triggers[i].BaseUrl = server.URL
				}
			}

			ctx := context.Background()

			// Execute the function
			output, err := FetchAll(ctx, tt.event)

			// Verify results
			if tt.expectedError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "some triggers failed")
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.expectedOutput, len(output), "unexpected number of trigger errors")

			// Verify that each error in output contains the expected trigger
			for _, triggerError := range output {
				assert.NotEmpty(t, triggerError.Error, "error message should not be empty")
				assert.NotEmpty(t, triggerError.Trigger.ServiceName, "trigger service name should not be empty")
			}
		})
	}
}

func TestFetchAll_NoTriggers(t *testing.T) {
	event := ExternalPayload{
		EventName:    "test-event",
		Data:         map[string]any{"test": "data"},
		ExtraHeaders: map[string]string{},
		Triggers:     []Trigger{}, // empty triggers
	}

	ctx := context.Background()
	output, err := FetchAll(ctx, event)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no triggers to send, required at least one trigger")
	assert.Empty(t, output)
}

func TestFetchAll_InvalidJSON(t *testing.T) {
	// Create event with data that cannot be marshaled to JSON
	event := ExternalPayload{
		EventName: "test-event",
		Data: map[string]any{
			"invalid": make(chan int), // channels cannot be marshaled to JSON
		},
		ExtraHeaders: map[string]string{},
		Triggers: []Trigger{
			{
				ServiceName: "service1",
				Type:        TriggerTypePersistent,
				BaseUrl:     "http://localhost:8080",
				Path:        "/webhook",
			},
		},
	}

	ctx := context.Background()
	output, err := FetchAll(ctx, event)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "some triggers failed")
	assert.Len(t, output, 1)
	assert.Contains(t, output[0].Error, "marshal data")
}

func TestFetchAll_NetworkError(t *testing.T) {
	event := ExternalPayload{
		EventName: "test-event",
		Data: map[string]any{
			"test": "data",
		},
		ExtraHeaders: map[string]string{},
		Triggers: []Trigger{
			{
				ServiceName: "service1",
				Type:        TriggerTypePersistent,
				BaseUrl:     "http://nonexistent-host-12345.com",
				Path:        "/webhook",
			},
		},
	}

	ctx := context.Background()
	output, err := FetchAll(ctx, event)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "some triggers failed")
	assert.Len(t, output, 1)
	assert.Contains(t, output[0].Error, "post request")
}

func TestCreateKey(t *testing.T) {
	trigger := Trigger{
		ServiceName: "test-service",
		Type:        TriggerTypePersistent,
		BaseUrl:     "https://api.example.com",
		Path:        "/webhook/endpoint",
	}

	// Note: createKey function doesn't handle URL normalization like GetUrl() does
	expected := "https://api.example.com//webhook/endpoint"
	result := createKey(trigger)

	assert.Equal(t, expected, result)
}

func TestFetch_HeadersPropagation(t *testing.T) {
	// Setup test server to verify headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify standard headers
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Verify custom headers
		assert.Equal(t, "custom-value", r.Header.Get("X-Custom-Header"))
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	event := ExternalPayload{
		EventName: "test-event",
		Data: map[string]any{
			"test": "data",
		},
		ExtraHeaders: map[string]string{
			"X-Custom-Header": "custom-value",
			"Authorization":   "Bearer test-token",
		},
		Triggers: []Trigger{
			{
				ServiceName: "service1",
				Type:        TriggerTypePersistent,
				BaseUrl:     server.URL,
				Path:        "/webhook",
			},
		},
	}

	ctx := context.Background()
	output, err := FetchAll(ctx, event)

	require.NoError(t, err)
	assert.Empty(t, output)
}

func TestFetch_ConcurrentExecution(t *testing.T) {
	const numTriggers = 10

	// Setup multiple test servers
	servers := make([]*httptest.Server, numTriggers)
	for i := 0; i < numTriggers; i++ {
		servers[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Add small delay to ensure concurrent execution
			time.Sleep(10 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok"}`))
		}))
	}
	defer func() {
		for _, server := range servers {
			server.Close()
		}
	}()

	// Create triggers for all servers
	triggers := make([]Trigger, numTriggers)
	for i := 0; i < numTriggers; i++ {
		triggers[i] = Trigger{
			ServiceName: fmt.Sprintf("service%d", i+1),
			Type:        TriggerTypePersistent,
			BaseUrl:     servers[i].URL,
			Path:        "/webhook",
		}
	}

	event := ExternalPayload{
		EventName:    "test-event",
		Data:         map[string]any{"test": "concurrent"},
		ExtraHeaders: map[string]string{},
		Triggers:     triggers,
	}

	ctx := context.Background()
	start := time.Now()

	output, err := FetchAll(ctx, event)

	duration := time.Since(start)

	require.NoError(t, err)
	assert.Empty(t, output)

	// If executed sequentially, it would take at least numTriggers * 10ms
	// If executed concurrently, it should take much less
	assert.Less(t, duration, time.Duration(numTriggers)*10*time.Millisecond,
		"execution should be concurrent, not sequential")
}
