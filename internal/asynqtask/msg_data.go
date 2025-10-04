package asynqtask

import (
	"encoding/json"
	"fmt"
)

type RawMsg string

func (m RawMsg) Msg() map[string]any {
	return unmarshal(string(m))
}

func unmarshal(raw string) map[string]any {
	var msg map[string]any

	err := json.Unmarshal([]byte(extractJSON(raw)), &msg)
	if err != nil {
		fmt.Println("Erro ao decodificar JSON:", err)
		return nil
	}

	return msg
}

func extractJSON(s string) string {
	start := -1
	for i, r := range s {
		if r == '{' {
			start = i
			break
		}
	}

	if start == -1 {
		return ""
	}

	braceCount := 0
	inString := false
	escaped := false

	for i := start; i < len(s); i++ {
		ch := s[i]

		if !inString {
			switch ch {
			case '{':
				braceCount++
			case '}':
				braceCount--
				if braceCount == 0 {
					return s[start : i+1]
				}
			case '"':
				inString = true
			}
		} else {
			if escaped {
				escaped = false
			} else if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				inString = false
			}
		}
	}

	return ""
}
