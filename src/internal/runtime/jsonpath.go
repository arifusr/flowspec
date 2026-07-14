package runtime

import (
	"strconv"
	"strings"
)

// evalJSONPath evaluates a simple JSONPath expression against a parsed JSON value.
// Supports: $, $.field, $.field.nested, $[0], $.field[0].nested
func evalJSONPath(data interface{}, path string) (interface{}, bool) {
	if path == "$" {
		return data, data != nil
	}

	// Remove leading "$." or "$"
	path = strings.TrimPrefix(path, "$.")
	path = strings.TrimPrefix(path, "$")

	parts := splitJSONPath(path)
	current := data

	for _, part := range parts {
		if current == nil {
			return nil, false
		}

		// Array index: [0], [1], etc.
		if strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]") {
			indexStr := part[1 : len(part)-1]
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				return nil, false
			}
			arr, ok := current.([]interface{})
			if !ok || index >= len(arr) || index < 0 {
				return nil, false
			}
			current = arr[index]
			continue
		}

		// Check if part has array index: field[0]
		if idx := strings.Index(part, "["); idx != -1 {
			field := part[:idx]
			indexPart := part[idx:]
			// Get field first
			m, ok := current.(map[string]interface{})
			if !ok {
				return nil, false
			}
			val, exists := m[field]
			if !exists {
				return nil, false
			}
			// Then index
			indexStr := indexPart[1 : len(indexPart)-1]
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				return nil, false
			}
			arr, ok := val.([]interface{})
			if !ok || index >= len(arr) || index < 0 {
				return nil, false
			}
			current = arr[index]
			continue
		}

		// Object field
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		val, exists := m[part]
		if !exists {
			return nil, false
		}
		current = val
	}

	return current, true
}

// splitJSONPath splits a path like "data.items[0].name" into parts.
func splitJSONPath(path string) []string {
	var parts []string
	var current strings.Builder

	for i := 0; i < len(path); i++ {
		ch := path[i]
		switch ch {
		case '.':
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		case '[':
			if current.Len() > 0 {
				// Include the bracket part with the field
				field := current.String()
				current.Reset()
				// Find closing bracket
				j := i + 1
				for j < len(path) && path[j] != ']' {
					j++
				}
				if j < len(path) {
					parts = append(parts, field+path[i:j+1])
					i = j
				}
			} else {
				// standalone [0]
				j := i + 1
				for j < len(path) && path[j] != ']' {
					j++
				}
				if j < len(path) {
					parts = append(parts, path[i:j+1])
					i = j
				}
			}
		default:
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}
