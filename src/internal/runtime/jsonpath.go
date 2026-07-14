package runtime

import (
	"fmt"
	"strconv"
	"strings"
)

// evalJSONPath evaluates a JSONPath expression against a parsed JSON value.
// Supports: $, $.field, $.field.nested, $[0], $.field[0].nested,
// $[?(@.field=='value')], $.data[?(@.name=='ABC')].id
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

		// Filter expression: [?(@.field=='value')] or [?(@.field==value)]
		if strings.HasPrefix(part, "[?(") && strings.HasSuffix(part, ")]") {
			current = evalFilter(current, part)
			if current == nil {
				return nil, false
			}
			continue
		}

		// Check if part contains a filter: field[?(@.x=='y')]
		if filterIdx := strings.Index(part, "[?("); filterIdx != -1 {
			field := part[:filterIdx]
			filterPart := part[filterIdx:]

			// Get the field value first
			if field != "" {
				m, ok := current.(map[string]interface{})
				if !ok {
					return nil, false
				}
				val, exists := m[field]
				if !exists {
					return nil, false
				}
				current = val
			}

			// Apply filter
			current = evalFilter(current, filterPart)
			if current == nil {
				return nil, false
			}
			continue
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

// evalFilter evaluates a filter expression like [?(@.name=='value')]
// Returns the first matching item from an array, or nil if no match.
func evalFilter(data interface{}, filter string) interface{} {
	arr, ok := data.([]interface{})
	if !ok {
		return nil
	}

	// Parse filter: [?(@.field=='value')] or [?(@.field==value)]
	inner := filter[3 : len(filter)-2] // strip [?( and )]
	// inner is now: @.field=='value' or @.field==value

	// Split by operator
	var field, operator, value string
	for _, op := range []string{"==", "!=", ">=", "<=", ">", "<"} {
		if idx := strings.Index(inner, op); idx != -1 {
			field = strings.TrimSpace(inner[:idx])
			operator = op
			value = strings.TrimSpace(inner[idx+len(op):])
			break
		}
	}

	if field == "" {
		return nil
	}

	// Strip @. prefix from field
	field = strings.TrimPrefix(field, "@.")

	// Strip quotes from value
	value = strings.Trim(value, "'\"")

	// Find matching items
	var matches []interface{}
	for _, item := range arr {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		// Support nested fields: @.data.name
		var itemVal interface{}
		if strings.Contains(field, ".") {
			itemVal, _ = evalJSONPath(m, "$."+field)
		} else {
			itemVal = m[field]
		}

		if itemVal == nil {
			continue
		}

		itemStr := fmt.Sprintf("%v", itemVal)

		switch operator {
		case "==":
			if itemStr == value {
				matches = append(matches, item)
			}
		case "!=":
			if itemStr != value {
				matches = append(matches, item)
			}
		case ">":
			if compareNumeric(itemStr, value) > 0 {
				matches = append(matches, item)
			}
		case ">=":
			if compareNumeric(itemStr, value) >= 0 {
				matches = append(matches, item)
			}
		case "<":
			if compareNumeric(itemStr, value) < 0 {
				matches = append(matches, item)
			}
		case "<=":
			if compareNumeric(itemStr, value) <= 0 {
				matches = append(matches, item)
			}
		}
	}

	if len(matches) == 0 {
		return nil
	}
	if len(matches) == 1 {
		return matches[0]
	}
	return matches
}

func compareNumeric(a, b string) int {
	af, errA := strconv.ParseFloat(a, 64)
	bf, errB := strconv.ParseFloat(b, 64)
	if errA != nil || errB != nil {
		return strings.Compare(a, b)
	}
	if af > bf {
		return 1
	}
	if af < bf {
		return -1
	}
	return 0
}

// splitJSONPath splits a path like "data.items[0].name" into parts.
// Handles filter expressions like data[?(@.name=='x')].id without splitting inside [?()]
func splitJSONPath(path string) []string {
	var parts []string
	var current strings.Builder
	depth := 0 // track bracket depth

	for i := 0; i < len(path); i++ {
		ch := path[i]
		switch {
		case ch == '[':
			depth++
			if current.Len() == 0 {
				// standalone bracket expression
				current.WriteByte(ch)
			} else {
				// bracket attached to field: field[...]
				current.WriteByte(ch)
			}
		case ch == ']':
			depth--
			current.WriteByte(ch)
			if depth == 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		case ch == '.' && depth == 0:
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
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
