package queryparser

import (
	"net/url"
	"reflect"
	"testing"
)

type TestStruct struct {
	Name    string   `query:"name"`
	Age     uint     `query:"age"`
	Tags    []string `query:"tags"`
	Active  bool     `query:"active"`
	Score   float64  `query:"score"`
	Numbers []int    `query:"numbers"`
}

func TestParseQueryParams(t *testing.T) {
	tests := []struct {
		name        string
		queryString string
		expected    TestStruct
		wantError   bool
	}{
		{
			name:        "basic parsing",
			queryString: "name=john&age=25&active=true&score=95.5",
			expected: TestStruct{
				Name:   "john",
				Age:    25,
				Active: true,
				Score:  95.5,
			},
		},
		{
			name:        "slice parsing with multiple values",
			queryString: "tags=go&tags=programming&tags=web",
			expected: TestStruct{
				Tags: []string{"go", "programming", "web"},
			},
		},
		{
			name:        "slice parsing with comma-separated values",
			queryString: "tags=go,programming,web",
			expected: TestStruct{
				Tags: []string{"go", "programming", "web"},
			},
		},
		{
			name:        "mixed slice parsing",
			queryString: "tags=go,programming&tags=web&tags=backend,frontend",
			expected: TestStruct{
				Tags: []string{"go", "programming", "web", "backend", "frontend"},
			},
		},
		{
			name:        "number slice parsing",
			queryString: "numbers=1,2,3&numbers=4&numbers=5,6",
			expected: TestStruct{
				Numbers: []int{1, 2, 3, 4, 5, 6},
			},
		},
		{
			name:        "empty values should be ignored",
			queryString: "tags=go,,programming&tags=&tags=web",
			expected: TestStruct{
				Tags: []string{"go", "programming", "web"},
			},
		},
		{
			name:        "all types combined",
			queryString: "name=alice&age=30&tags=golang,api&active=false&score=88.7&numbers=10,20",
			expected: TestStruct{
				Name:    "alice",
				Age:     30,
				Tags:    []string{"golang", "api"},
				Active:  false,
				Score:   88.7,
				Numbers: []int{10, 20},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, err := url.ParseQuery(tt.queryString)
			if err != nil {
				t.Fatalf("Failed to parse query string: %v", err)
			}

			var result TestStruct
			err = ParseQueryParams(values, &result)

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}

func TestParseQueryParamsWithDefaults(t *testing.T) {
	queryString := "name=bob"
	values, _ := url.ParseQuery(queryString)

	defaults := map[string]interface{}{
		"age":    uint(18),
		"active": true,
		"score":  float64(0.0),
		"tags":   []string{"default"},
	}

	var result TestStruct
	err := ParseQueryParamsWithDefaults(values, &result, defaults)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := TestStruct{
		Name:   "bob",
		Age:    18,
		Active: true,
		Score:  0.0,
		Tags:   []string{"default"},
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %+v, got %+v", expected, result)
	}
}

func TestParseQueryParamsErrors(t *testing.T) {
	tests := []struct {
		name        string
		target      interface{}
		queryString string
		wantError   bool
	}{
		{
			name:      "non-pointer target",
			target:    TestStruct{},
			wantError: true,
		},
		{
			name:      "nil target",
			target:    (*TestStruct)(nil),
			wantError: true,
		},
		{
			name:        "invalid integer",
			target:      &TestStruct{},
			queryString: "age=invalid",
			wantError:   true,
		},
		{
			name:        "invalid boolean",
			target:      &TestStruct{},
			queryString: "active=maybe",
			wantError:   true,
		},
		{
			name:        "invalid float",
			target:      &TestStruct{},
			queryString: "score=not_a_number",
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, _ := url.ParseQuery(tt.queryString)
			err := ParseQueryParams(values, tt.target)

			if tt.wantError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// FilterEvents test to match your domain struct
type FilterEvents struct {
	TeamOwner   []string `query:"team_owner"`
	ServiceName []string `query:"service_name"`
	State       []string `query:"state"`
	Page        uint     `query:"page"`
	Limit       uint     `query:"limit"`
}

func TestFilterEventsExample(t *testing.T) {
	// Simulate a real query string like your API would receive
	queryString := "service_name=user-service,order-service&state=active&team_owner=backend&page=1&limit=50"
	values, _ := url.ParseQuery(queryString)

	var filter FilterEvents
	err := ParseQueryParams(values, &filter)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := FilterEvents{
		ServiceName: []string{"user-service", "order-service"},
		State:       []string{"active"},
		TeamOwner:   []string{"backend"},
		Page:        1,
		Limit:       50,
	}

	if !reflect.DeepEqual(filter, expected) {
		t.Errorf("Expected %+v, got %+v", expected, filter)
	}
}

func TestFilterEventsWithDefaults(t *testing.T) {
	// Test with minimal query parameters
	queryString := "service_name=api-gateway"
	values, _ := url.ParseQuery(queryString)

	defaults := map[string]interface{}{
		"page":  uint(1),
		"limit": uint(100),
		"state": []string{"active"},
	}

	var filter FilterEvents
	err := ParseQueryParamsWithDefaults(values, &filter, defaults)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := FilterEvents{
		ServiceName: []string{"api-gateway"},
		State:       []string{"active"},
		Page:        1,
		Limit:       100,
	}

	if !reflect.DeepEqual(filter, expected) {
		t.Errorf("Expected %+v, got %+v", expected, filter)
	}
}
