package runtime

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Variables manages variable scoping and resolution.
type Variables struct {
	scopes []map[string]string
}

// NewVariables creates a new Variables with a global scope.
func NewVariables() *Variables {
	return &Variables{
		scopes: []map[string]string{make(map[string]string)},
	}
}

// PushScope pushes a new variable scope.
func (v *Variables) PushScope() {
	v.scopes = append(v.scopes, make(map[string]string))
}

// PopScope removes the top variable scope.
func (v *Variables) PopScope() {
	if len(v.scopes) > 1 {
		v.scopes = v.scopes[:len(v.scopes)-1]
	}
}

// Set sets a variable in the current (top) scope.
func (v *Variables) Set(key, value string) {
	v.scopes[len(v.scopes)-1][key] = value
}

// Get retrieves a variable by searching scopes from top to bottom.
func (v *Variables) Get(key string) (string, bool) {
	for i := len(v.scopes) - 1; i >= 0; i-- {
		if val, ok := v.scopes[i][key]; ok {
			return val, true
		}
	}
	return "", false
}

// Has checks if variable exists.
func (v *Variables) Has(key string) bool {
	_, ok := v.Get(key)
	return ok
}

// All returns all variables merged (lower scopes first).
func (v *Variables) All() map[string]string {
	result := make(map[string]string)
	for _, scope := range v.scopes {
		for k, val := range scope {
			result[k] = val
		}
	}
	return result
}

var interpolateRe = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// Interpolate replaces {{var}} placeholders in a string.
func (v *Variables) Interpolate(s string) string {
	return interpolateRe.ReplaceAllStringFunc(s, func(match string) string {
		key := strings.TrimSpace(match[2 : len(match)-2])

		// Built-in dynamic variables
		if strings.HasPrefix(key, "$") {
			return v.resolveDynamic(key)
		}

		if val, ok := v.Get(key); ok {
			return val
		}
		return match // leave unresolved
	})
}

func (v *Variables) resolveDynamic(key string) string {
	switch key {
	case "$uuid":
		return uuid.New().String()
	case "$timestamp":
		return fmt.Sprintf("%d", time.Now().Unix())
	case "$randomInt":
		return fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
	case "$randomEmail":
		return fmt.Sprintf("test-%s@example.com", uuid.New().String()[:8])
	default:
		// $env.VAR_NAME
		if strings.HasPrefix(key, "$env.") {
			envKey := key[5:]
			return os.Getenv(envKey)
		}
		return "{{" + key + "}}"
	}
}
