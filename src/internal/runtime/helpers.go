package runtime

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// parseDuration parses duration strings like "500ms", "2s", "1m".
func parseDuration(s string) time.Duration {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "ms") {
		n, _ := strconv.Atoi(strings.TrimSuffix(s, "ms"))
		return time.Duration(n) * time.Millisecond
	}
	if strings.HasSuffix(s, "s") {
		n, _ := strconv.Atoi(strings.TrimSuffix(s, "s"))
		return time.Duration(n) * time.Second
	}
	if strings.HasSuffix(s, "m") {
		n, _ := strconv.Atoi(strings.TrimSuffix(s, "m"))
		return time.Duration(n) * time.Minute
	}
	if strings.HasSuffix(s, "h") {
		n, _ := strconv.Atoi(strings.TrimSuffix(s, "h"))
		return time.Duration(n) * time.Hour
	}
	// try Go duration parse
	d, _ := time.ParseDuration(s)
	return d
}

// parseSize parses size strings like "100bytes", "1kb", "1mb".
func parseSize(s string) int64 {
	s = strings.TrimSpace(strings.ToLower(s))
	if strings.HasSuffix(s, "gb") {
		n, _ := strconv.ParseInt(strings.TrimSuffix(s, "gb"), 10, 64)
		return n * 1024 * 1024 * 1024
	}
	if strings.HasSuffix(s, "mb") {
		n, _ := strconv.ParseInt(strings.TrimSuffix(s, "mb"), 10, 64)
		return n * 1024 * 1024
	}
	if strings.HasSuffix(s, "kb") {
		n, _ := strconv.ParseInt(strings.TrimSuffix(s, "kb"), 10, 64)
		return n * 1024
	}
	if strings.HasSuffix(s, "bytes") {
		n, _ := strconv.ParseInt(strings.TrimSuffix(s, "bytes"), 10, 64)
		return n
	}
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

// getJSONType returns the type name of a JSON value.
func getJSONType(v interface{}) string {
	switch v.(type) {
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	case float64:
		return "number"
	case bool:
		return "boolean"
	case string:
		return "string"
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%T", v)
	}
}

// getLength returns the length of an array or string.
func getLength(v interface{}) int {
	switch val := v.(type) {
	case []interface{}:
		return len(val)
	case string:
		return len(val)
	case map[string]interface{}:
		return len(val)
	default:
		return 0
	}
}

// toFloat converts a JSON value to float64.
func toFloat(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		return 0
	}
}
