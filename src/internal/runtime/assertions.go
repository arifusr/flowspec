package runtime

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/testing-cli/apitest/internal/ast"
)

// AssertionResult represents the result of a single assertion.
type AssertionResult struct {
	Passed   bool
	Message  string
	Expected string
	Actual   string
	Soft     bool
}

// EvalExpect evaluates an expect declaration against a response.
func EvalExpect(exp *ast.ExpectDecl, resp *HTTPResponse, vars *Variables) AssertionResult {
	switch exp.Type {
	case "status":
		return evalStatusExpect(exp, resp)
	case "json":
		return evalJSONExpect(exp, resp, vars)
	case "header":
		return evalHeaderExpect(exp, resp, vars)
	case "time":
		return evalTimeExpect(exp, resp)
	case "size":
		return evalSizeExpect(exp, resp)
	default:
		return AssertionResult{
			Passed:  false,
			Message: fmt.Sprintf("unknown expect type: %s", exp.Type),
			Soft:    exp.Soft,
		}
	}
}

func evalStatusExpect(exp *ast.ExpectDecl, resp *HTTPResponse) AssertionResult {
	result := AssertionResult{Soft: exp.Soft}

	if exp.StatusRange != "" {
		// e.g. "2xx" means 200-299
		prefix := exp.StatusRange[0] - '0'
		low := int(prefix) * 100
		high := low + 99
		result.Expected = exp.StatusRange
		result.Actual = strconv.Itoa(resp.StatusCode)
		result.Passed = resp.StatusCode >= low && resp.StatusCode <= high
		if exp.Negated {
			result.Passed = !result.Passed
		}
	} else if len(exp.StatusCodes) > 0 {
		result.Expected = fmt.Sprintf("one of %v", exp.StatusCodes)
		result.Actual = strconv.Itoa(resp.StatusCode)
		for _, code := range exp.StatusCodes {
			if resp.StatusCode == code {
				result.Passed = true
				break
			}
		}
	} else {
		result.Expected = strconv.Itoa(exp.StatusCode)
		result.Actual = strconv.Itoa(resp.StatusCode)
		if exp.Negated {
			result.Passed = resp.StatusCode != exp.StatusCode
		} else {
			result.Passed = resp.StatusCode == exp.StatusCode
		}
	}

	if !result.Passed {
		op := "=="
		if exp.Negated {
			op = "!="
		}
		result.Message = fmt.Sprintf("expect status %s %s, got %s",
			op, result.Expected, result.Actual)
	} else {
		result.Message = fmt.Sprintf("expect status %s", result.Actual)
	}
	return result
}

func evalJSONExpect(exp *ast.ExpectDecl, resp *HTTPResponse, vars *Variables) AssertionResult {
	result := AssertionResult{Soft: exp.Soft}
	jsonPath := exp.JSONPath
	expectedVal := vars.Interpolate(exp.Value)

	// Parse response body
	var body interface{}
	if err := json.Unmarshal(resp.Body, &body); err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("failed to parse response as JSON: %s", err)
		return result
	}

	// Evaluate JSONPath
	actual, found := evalJSONPath(body, jsonPath)

	switch exp.Operator {
	case "exists":
		result.Passed = found != exp.Negated
		result.Expected = "exists"
		if exp.Negated {
			result.Expected = "not exists"
		}
		result.Actual = fmt.Sprintf("found=%v", found)
		if !result.Passed {
			result.Message = fmt.Sprintf("expect json %q %s", jsonPath, result.Expected)
		}

	case "==":
		actualStr := fmt.Sprintf("%v", actual)
		result.Expected = expectedVal
		result.Actual = actualStr
		result.Passed = actualStr == expectedVal
		if !result.Passed {
			result.Message = fmt.Sprintf("expect json %q == %q, got %q",
				jsonPath, expectedVal, actualStr)
		}

	case "!=":
		actualStr := fmt.Sprintf("%v", actual)
		result.Expected = "!= " + expectedVal
		result.Actual = actualStr
		result.Passed = actualStr != expectedVal
		if !result.Passed {
			result.Message = fmt.Sprintf("expect json %q != %q, got %q",
				jsonPath, expectedVal, actualStr)
		}

	case "is":
		actualType := getJSONType(actual)
		result.Expected = expectedVal
		result.Actual = actualType
		result.Passed = actualType == expectedVal
		if !result.Passed {
			result.Message = fmt.Sprintf("expect json %q is %s, got %s",
				jsonPath, expectedVal, actualType)
		}

	case "length":
		length := getLength(actual)
		expectedLen, _ := strconv.Atoi(expectedVal)
		result.Expected = expectedVal
		result.Actual = strconv.Itoa(length)
		result.Passed = length == expectedLen
		if !result.Passed {
			result.Message = fmt.Sprintf("expect json %q length %d, got %d",
				jsonPath, expectedLen, length)
		}

	case "length>=":
		length := getLength(actual)
		expectedLen, _ := strconv.Atoi(expectedVal)
		result.Expected = ">= " + expectedVal
		result.Actual = strconv.Itoa(length)
		result.Passed = length >= expectedLen

	case "length<=":
		length := getLength(actual)
		expectedLen, _ := strconv.Atoi(expectedVal)
		result.Expected = "<= " + expectedVal
		result.Actual = strconv.Itoa(length)
		result.Passed = length <= expectedLen

	case "length>":
		length := getLength(actual)
		expectedLen, _ := strconv.Atoi(expectedVal)
		result.Passed = length > expectedLen

	case "length<":
		length := getLength(actual)
		expectedLen, _ := strconv.Atoi(expectedVal)
		result.Passed = length < expectedLen

	case ">=", "<=", ">", "<":
		actualNum := toFloat(actual)
		expectedNum, _ := strconv.ParseFloat(expectedVal, 64)
		result.Expected = exp.Operator + " " + expectedVal
		result.Actual = fmt.Sprintf("%v", actual)
		switch exp.Operator {
		case ">=":
			result.Passed = actualNum >= expectedNum
		case "<=":
			result.Passed = actualNum <= expectedNum
		case ">":
			result.Passed = actualNum > expectedNum
		case "<":
			result.Passed = actualNum < expectedNum
		}

	case "matches":
		actualStr := fmt.Sprintf("%v", actual)
		re, err := regexp.Compile(expectedVal)
		if err != nil {
			result.Passed = false
			result.Message = fmt.Sprintf("invalid regex %q: %s", expectedVal, err)
			return result
		}
		result.Passed = re.MatchString(actualStr)
		result.Expected = "matches " + expectedVal
		result.Actual = actualStr

	case "contains":
		actualStr := fmt.Sprintf("%v", actual)
		result.Passed = strings.Contains(actualStr, expectedVal)
		result.Expected = "contains " + expectedVal
		result.Actual = actualStr
	}

	if result.Message == "" && !result.Passed {
		result.Message = fmt.Sprintf("expect json %q %s %s — actual: %s",
			jsonPath, exp.Operator, result.Expected, result.Actual)
	}
	if result.Message == "" {
		result.Message = fmt.Sprintf("expect json %q %s %s", jsonPath, exp.Operator, expectedVal)
	}
	return result
}

func evalHeaderExpect(exp *ast.ExpectDecl, resp *HTTPResponse, vars *Variables) AssertionResult {
	result := AssertionResult{Soft: exp.Soft}
	headerName := exp.HeaderName
	expectedVal := vars.Interpolate(exp.Value)
	actualVal := resp.Headers.Get(headerName)

	switch exp.Operator {
	case "exists":
		result.Passed = actualVal != ""
		result.Expected = "exists"
		result.Actual = actualVal
		if !result.Passed {
			result.Message = fmt.Sprintf("expect header %q exists, but not found", headerName)
		}
	case "==":
		result.Passed = actualVal == expectedVal
		result.Expected = expectedVal
		result.Actual = actualVal
	case "contains":
		result.Passed = strings.Contains(actualVal, expectedVal)
		result.Expected = "contains " + expectedVal
		result.Actual = actualVal
	case "matches":
		re, _ := regexp.Compile(expectedVal)
		result.Passed = re != nil && re.MatchString(actualVal)
		result.Expected = "matches " + expectedVal
		result.Actual = actualVal
	}

	if result.Message == "" && !result.Passed {
		result.Message = fmt.Sprintf("expect header %q %s %q, got %q",
			headerName, exp.Operator, expectedVal, actualVal)
	}
	if result.Message == "" {
		result.Message = fmt.Sprintf("expect header %q %s", headerName, exp.Operator)
	}
	return result
}

func evalTimeExpect(exp *ast.ExpectDecl, resp *HTTPResponse) AssertionResult {
	result := AssertionResult{Soft: exp.Soft}
	maxDuration := parseDuration(exp.Duration)
	actual := resp.Duration

	result.Expected = exp.Operator + " " + exp.Duration
	result.Actual = actual.String()

	switch exp.Operator {
	case "<":
		result.Passed = actual < maxDuration
	case "<=":
		result.Passed = actual <= maxDuration
	}

	if !result.Passed {
		result.Message = fmt.Sprintf("expect time %s %s, got %s",
			exp.Operator, exp.Duration, actual)
	} else {
		result.Message = fmt.Sprintf("expect time %s %s", exp.Operator, exp.Duration)
	}
	return result
}

func evalSizeExpect(exp *ast.ExpectDecl, resp *HTTPResponse) AssertionResult {
	result := AssertionResult{Soft: exp.Soft}
	maxSize := parseSize(exp.Size)
	actual := resp.Size

	result.Expected = exp.Operator + " " + exp.Size
	result.Actual = fmt.Sprintf("%d bytes", actual)

	switch exp.Operator {
	case "<":
		result.Passed = actual < maxSize
	case "<=":
		result.Passed = actual <= maxSize
	case ">":
		result.Passed = actual > maxSize
	case ">=":
		result.Passed = actual >= maxSize
	}

	if !result.Passed {
		result.Message = fmt.Sprintf("expect size %s %s, got %d bytes",
			exp.Operator, exp.Size, actual)
	}
	return result
}
