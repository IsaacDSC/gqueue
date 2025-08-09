package intertime

import (
	"encoding/json"
	"time"
)

// Duration is a custom type that wraps time.Duration to provide JSON
// marshaling and unmarshaling capabilities. It allows durations to be
// represented as human-readable strings (e.g., "1h30m", "500ms") in JSON
// instead of nanosecond integers.
//
// Example usage:
//
//	type Config struct {
//		Timeout Duration `json:"timeout"`
//	}
//
//	// JSON: {"timeout": "30s"}
type Duration time.Duration

// UnmarshalJSON implements the json.Unmarshaler interface.
// It parses a JSON string representation of a duration (e.g., "1h30m", "500ms")
// and converts it to a Duration value.
//
// The string must be a valid duration string as accepted by time.ParseDuration.
// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
//

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	duration, err := time.ParseDuration(s)
	if err != nil {
		return err
	}

	*d = Duration(duration)
	return nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d Duration) String() string {
	return time.Duration(d).String()
}
