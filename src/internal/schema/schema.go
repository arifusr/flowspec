package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Schema represents a parsed JSON Schema.
type Schema struct {
	ID         string             `json:"$id,omitempty"`
	Ref        string             `json:"$ref,omitempty"`
	Type       string             `json:"type"`
	Properties map[string]*Schema `json:"properties,omitempty"`
	Items      *Schema            `json:"items,omitempty"`
	Required   []string           `json:"required,omitempty"`
	Default    interface{}        `json:"default,omitempty"`
	Example    interface{}        `json:"example,omitempty"`
	Enum       []interface{}      `json:"enum,omitempty"`
	Format     string             `json:"format,omitempty"`
}

// LoadSchema reads and parses a JSON Schema file.
// Resolves $ref relative to the schema file's directory.
func LoadSchema(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read schema %s: %w", path, err)
	}

	var s Schema
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("invalid JSON schema %s: %w", path, err)
	}

	// Resolve $ref in properties and items
	dir := filepath.Dir(path)
	resolveRefs(&s, dir)

	return &s, nil
}

// resolveRefs recursively resolves $ref fields by loading referenced schemas.
func resolveRefs(s *Schema, baseDir string) {
	if s == nil {
		return
	}

	for key, prop := range s.Properties {
		if prop.Ref != "" {
			refPath := filepath.Join(baseDir, prop.Ref)
			refSchema, err := LoadSchema(refPath)
			if err == nil {
				s.Properties[key] = refSchema
			}
		}
		resolveRefs(prop, baseDir)
	}

	if s.Items != nil {
		if s.Items.Ref != "" {
			refPath := filepath.Join(baseDir, s.Items.Ref)
			refSchema, err := LoadSchema(refPath)
			if err == nil {
				s.Items = refSchema
			}
		}
		resolveRefs(s.Items, baseDir)
	}
}

// GeneratePayload generates a sample JSON object from a schema using default/example values.
func GeneratePayload(s *Schema) interface{} {
	if s == nil {
		return nil
	}

	switch s.Type {
	case "object":
		obj := make(map[string]interface{})
		for key, prop := range s.Properties {
			obj[key] = GeneratePayload(prop)
		}
		return obj

	case "array":
		if s.Items != nil {
			// Generate one sample item
			return []interface{}{GeneratePayload(s.Items)}
		}
		return []interface{}{}

	case "string":
		if s.Example != nil {
			return fmt.Sprintf("%v", s.Example)
		}
		if s.Default != nil {
			return fmt.Sprintf("%v", s.Default)
		}
		if len(s.Enum) > 0 {
			return fmt.Sprintf("%v", s.Enum[0])
		}
		return ""

	case "integer", "number":
		if s.Example != nil {
			return s.Example
		}
		if s.Default != nil {
			return s.Default
		}
		return 0

	case "boolean":
		if s.Example != nil {
			return s.Example
		}
		if s.Default != nil {
			return s.Default
		}
		return false

	default:
		if s.Example != nil {
			return s.Example
		}
		if s.Default != nil {
			return s.Default
		}
		return nil
	}
}

// ApplyOverrides applies `set` overrides to a generated payload.
// Supports deep path like: "items[0].components[0].qty"
func ApplyOverrides(payload interface{}, overrides map[string]string) interface{} {
	for path, value := range overrides {
		payload = setDeepValue(payload, path, value)
	}
	return payload
}

// setDeepValue sets a value at a deep path in a nested map/slice structure.
func setDeepValue(data interface{}, path string, value string) interface{} {
	parts := splitSetPath(path)
	return setRecursive(data, parts, value)
}

func setRecursive(data interface{}, parts []pathPart, value string) interface{} {
	if len(parts) == 0 {
		return parseSetValue(value)
	}

	current := parts[0]
	rest := parts[1:]

	if current.isIndex {
		// Array index access
		arr, ok := data.([]interface{})
		if !ok {
			arr = []interface{}{}
		}
		// Extend array if needed
		for len(arr) <= current.index {
			arr = append(arr, make(map[string]interface{}))
		}
		arr[current.index] = setRecursive(arr[current.index], rest, value)
		return arr
	}

	// Object field access
	obj, ok := data.(map[string]interface{})
	if !ok {
		obj = make(map[string]interface{})
	}

	if len(rest) == 0 {
		obj[current.field] = parseSetValue(value)
	} else {
		existing, exists := obj[current.field]
		if !exists {
			// Determine if next part needs array or object
			if len(rest) > 0 && rest[0].isIndex {
				existing = []interface{}{}
			} else {
				existing = make(map[string]interface{})
			}
		}
		obj[current.field] = setRecursive(existing, rest, value)
	}
	return obj
}

type pathPart struct {
	field   string
	isIndex bool
	index   int
}

// splitSetPath splits "items[0].components[1].qty" into path parts.
func splitSetPath(path string) []pathPart {
	var parts []pathPart
	segments := strings.Split(path, ".")

	for _, seg := range segments {
		if idx := strings.Index(seg, "["); idx != -1 {
			// Field with index: items[0]
			field := seg[:idx]
			indexStr := seg[idx+1 : len(seg)-1]
			index, _ := strconv.Atoi(indexStr)

			if field != "" {
				parts = append(parts, pathPart{field: field})
			}
			parts = append(parts, pathPart{isIndex: true, index: index})
		} else {
			parts = append(parts, pathPart{field: seg})
		}
	}
	return parts
}

// parseSetValue converts a string value to appropriate JSON type.
func parseSetValue(value string) interface{} {
	// Empty array
	if value == "[]" {
		return []interface{}{}
	}
	// Empty object
	if value == "{}" {
		return map[string]interface{}{}
	}
	// Boolean
	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}
	// Integer
	if n, err := strconv.Atoi(value); err == nil {
		return n
	}
	// Float
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f
	}
	// String
	return value
}

// ValidateAgainstSchema validates a JSON response body against a schema.
// Returns a list of validation errors (empty = valid).
func ValidateAgainstSchema(data interface{}, s *Schema) []string {
	var errors []string
	validateRecursive(data, s, "$", &errors)
	return errors
}

func validateRecursive(data interface{}, s *Schema, path string, errors *[]string) {
	if s == nil || data == nil {
		return
	}

	switch s.Type {
	case "object":
		obj, ok := data.(map[string]interface{})
		if !ok {
			*errors = append(*errors, fmt.Sprintf("%s: expected object, got %T", path, data))
			return
		}
		// Check required fields
		for _, req := range s.Required {
			if _, exists := obj[req]; !exists {
				*errors = append(*errors, fmt.Sprintf("%s.%s: required field missing", path, req))
			}
		}
		// Validate properties
		for key, prop := range s.Properties {
			if val, exists := obj[key]; exists {
				validateRecursive(val, prop, path+"."+key, errors)
			}
		}

	case "array":
		arr, ok := data.([]interface{})
		if !ok {
			*errors = append(*errors, fmt.Sprintf("%s: expected array, got %T", path, data))
			return
		}
		if s.Items != nil {
			for i, item := range arr {
				validateRecursive(item, s.Items, fmt.Sprintf("%s[%d]", path, i), errors)
			}
		}

	case "string":
		if _, ok := data.(string); !ok {
			*errors = append(*errors, fmt.Sprintf("%s: expected string, got %T", path, data))
		}

	case "integer":
		switch data.(type) {
		case float64:
			// JSON numbers are float64, check if it's whole
			f := data.(float64)
			if f != float64(int(f)) {
				*errors = append(*errors, fmt.Sprintf("%s: expected integer, got float", path))
			}
		case int:
			// ok
		default:
			*errors = append(*errors, fmt.Sprintf("%s: expected integer, got %T", path, data))
		}

	case "number":
		switch data.(type) {
		case float64, int:
			// ok
		default:
			*errors = append(*errors, fmt.Sprintf("%s: expected number, got %T", path, data))
		}

	case "boolean":
		if _, ok := data.(bool); !ok {
			*errors = append(*errors, fmt.Sprintf("%s: expected boolean, got %T", path, data))
		}
	}
}
