package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/testing-cli/apitest/internal/ast"
	"github.com/testing-cli/apitest/internal/parser"
	"github.com/testing-cli/apitest/internal/schema"
)

// Engine executes FlowSpec AST nodes.
type Engine struct {
	Vars         *Variables
	Client       *HTTPClient
	Requests     map[string]*ast.RequestDecl
	Auths        map[string]*ast.AuthDecl
	Fragments    map[string]*ast.FragmentDecl
	Results      []StepResult
	LastResponse *HTTPResponse
	BaseDir      string
	Verbose      int
	FailFast     bool
	Timeout      time.Duration
	Redact       []string
}

// StepResult records the outcome of a single step.
type StepResult struct {
	Name        string
	RequestName string
	Method      string
	URL         string
	StatusCode  int
	Duration    time.Duration
	Assertions  []AssertionResult
	Passed      bool
	Skipped     bool
	Error       string
}

// NewEngine creates a new execution engine.
func NewEngine(baseDir string, timeout time.Duration, verbose int, failFast bool) *Engine {
	return &Engine{
		Vars:      NewVariables(),
		Client:    NewHTTPClient(timeout),
		Requests:  make(map[string]*ast.RequestDecl),
		Auths:     make(map[string]*ast.AuthDecl),
		Fragments: make(map[string]*ast.FragmentDecl),
		BaseDir:   baseDir,
		Verbose:   verbose,
		FailFast:  failFast,
		Timeout:   timeout,
	}
}

// LoadEnvFile loads an environment file and sets variables.
func (e *Engine) LoadEnvFile(path string, envName string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot read env file %s: %w", path, err)
	}

	p := parser.New(string(content), path)
	file := p.Parse()
	if len(p.Errors()) > 0 {
		return fmt.Errorf("parse error in %s: %s", path, p.Errors()[0].Error())
	}

	for _, env := range file.Envs {
		if envName == "" || env.Name == envName {
			for _, a := range env.Assignments {
				if a.IsEnvRef {
					e.Vars.Set(a.Key, os.Getenv(a.EnvVar))
				} else {
					e.Vars.Set(a.Key, a.Value)
				}
			}
		}
	}
	return nil
}

// LoadFile parses a .flow file and registers its definitions.
func (e *Engine) LoadFile(path string) (*ast.File, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read file %s: %w", path, err)
	}

	p := parser.New(string(content), path)
	file := p.Parse()
	if len(p.Errors()) > 0 {
		return nil, fmt.Errorf("parse errors in %s:\n%s", path, formatErrors(p.Errors()))
	}

	// Register definitions
	for i := range file.Requests {
		e.Requests[file.Requests[i].Name] = &file.Requests[i]
	}
	for i := range file.Auths {
		e.Auths[file.Auths[i].Name] = &file.Auths[i]
	}
	for i := range file.Fragments {
		e.Fragments[file.Fragments[i].Name] = &file.Fragments[i]
	}

	// Process imports
	for _, imp := range file.Imports {
		impPath := filepath.Join(e.BaseDir, imp.Path)
		if _, err := os.Stat(impPath); err == nil {
			_, _ = e.LoadFile(impPath)
		}
	}

	return file, nil
}

func formatErrors(errs []parser.ParseError) string {
	var msgs []string
	for _, e := range errs {
		msgs = append(msgs, e.Error())
	}
	return strings.Join(msgs, "\n")
}

// ExecuteRequest executes a single request declaration.
func (e *Engine) ExecuteRequest(req *ast.RequestDecl) StepResult {
	result := StepResult{Name: req.Name, RequestName: req.Name, Method: req.Method}

	// Resolve auth headers
	headers := make(map[string]string)
	if req.UseAuth != "" {
		if auth, ok := e.Auths[req.UseAuth]; ok {
			for _, h := range auth.Headers {
				headers[h.Key] = h.Value
			}
		}
	}
	for _, h := range req.Headers {
		headers[h.Key] = h.Value
	}

	// Resolve queries
	queries := make(map[string]string)
	for _, q := range req.Queries {
		queries[q.Key] = q.Value
	}

	// Resolve body
	bodyFields := make(map[string]string)
	bodyType := ""
	if req.Body != nil {
		bodyType = req.Body.Type
		for _, f := range req.Body.Fields {
			bodyFields[f.Key] = f.Value
		}
	}

	var schemaBody []byte
	if bodyType == "schema" && req.Body.SchemaPath != "" {
		schemaBody = e.buildSchemaBody(req.Body)
	}

	// Execute before hooks
	for _, hook := range req.BeforeHook {
		switch hook.Type {
		case "set":
			e.Vars.Set(hook.Key, e.Vars.Interpolate(hook.Value))
		case "log":
			fmt.Printf("  📋 before: %s\n", e.interpolateWithLast(hook.Value))
		}
	}

	httpReq := BuildRequest(req.Method, req.URL, headers, queries, bodyType, bodyFields, e.Vars, e.Timeout)
	if schemaBody != nil {
		httpReq.Body = schemaBody
		httpReq.Headers["Content-Type"] = "application/json"
	}
	result.Method = httpReq.Method
	result.URL = httpReq.URL

	// Send request
	resp, err := e.Client.Send(httpReq)
	if err != nil {
		result.Error = err.Error()
		result.Passed = false
		return result
	}

	result.StatusCode = resp.StatusCode
	result.Duration = resp.Duration

	// Store last response for last.json() / last.header()
	e.LastResponse = resp

	// Set last.* variables so they're accessible via {{last.status}}, {{last.body}}
	e.Vars.Set("last.status", fmt.Sprintf("%d", resp.StatusCode))
	e.Vars.Set("last.body", string(resp.Body))

	// Run assertions
	allPassed := true
	for i := range req.Expects {
		ar := EvalExpect(&req.Expects[i], resp, e.Vars)
		result.Assertions = append(result.Assertions, ar)
		if !ar.Passed && !ar.Soft {
			allPassed = false
		}
	}

	// Extract variables
	for _, ext := range req.Extracts {
		val := extractValue(ext, resp)
		if val != "" {
			e.Vars.Set(ext.Variable, val)
			if e.Verbose >= 2 {
				display := val
				if len(display) > 40 {
					display = display[:40] + "..."
				}
				fmt.Printf("    ✓ extract %s = %s\n", ext.Variable, display)
			}
		} else if e.Verbose >= 2 {
			fmt.Printf("    ⚠ extract %s = (empty)\n", ext.Variable)
		}
	}

	// Execute after hooks
	for _, hook := range req.AfterHook {
		switch hook.Type {
		case "set":
			e.Vars.Set(hook.Key, e.Vars.Interpolate(hook.Value))
		case "log":
			fmt.Printf("  📋 after: %s\n", e.interpolateWithLast(hook.Value))
		}
	}

	result.Passed = allPassed
	return result
}

func extractValue(ext ast.ExtractDecl, resp *HTTPResponse) string {
	switch ext.Source {
	case "json":
		var body interface{}
		if err := json.Unmarshal(resp.Body, &body); err != nil {
			return ""
		}
		val, found := evalJSONPath(body, ext.Path)
		if !found {
			return ""
		}
		return fmt.Sprintf("%v", val)
	case "header":
		return resp.Headers.Get(ext.Path)
	case "cookie":
		// Simple cookie extraction from Set-Cookie header
		cookies := resp.Headers.Values("Set-Cookie")
		for _, c := range cookies {
			if strings.Contains(c, ext.Path) {
				// Find the value between 'name=' and first ';'
				prefix := ext.Path + "="
				idx := strings.Index(c, prefix)
				if idx != -1 {
					valueStart := idx + len(prefix)
					rest := c[valueStart:]
					semiIdx := strings.Index(rest, ";")
					if semiIdx != -1 {
						return rest[:semiIdx]
					}
					return rest
				}
				// Fallback: split by first =
				parts := strings.SplitN(c, "=", 2)
				if len(parts) == 2 {
					return strings.Split(parts[1], ";")[0]
				}
			}
		}
		return ""
	}
	return ""
}

// ExecuteFlow executes a flow (scenario).
func (e *Engine) ExecuteFlow(flow *ast.FlowDecl) []StepResult {
	var allResults []StepResult

	// Set flow-level variables (but don't override CLI vars)
	e.Vars.PushScope()
	defer e.Vars.PopScope()

	for _, l := range flow.Lets {
		// Only set if not already set by a higher-priority source (CLI --var)
		if _, exists := e.Vars.Get(l.Name); !exists {
			e.Vars.Set(l.Name, e.Vars.Interpolate(l.Value))
		}
	}

	// Execute steps
	for _, step := range flow.Steps {
		if e.FailFast && hasFailure(allResults) {
			break
		}
		stepResults := e.executeStep(&step)
		allResults = append(allResults, stepResults...)
	}

	// Execute teardown
	if flow.Teardown != nil {
		e.executeTeardown(flow.Teardown)
	}

	return allResults
}

func (e *Engine) executeStep(step *ast.StepDecl) []StepResult {
	var results []StepResult

	// Check when condition
	if step.When != "" {
		cond := strings.TrimSpace(step.When)
		if !e.evalCondition(cond) {
			results = append(results, StepResult{
				Name:    step.Name,
				Skipped: true,
				Passed:  true,
			})
			return results
		}
	}

	// Check unless condition
	if step.Unless != "" {
		cond := strings.TrimSpace(step.Unless)
		if e.evalCondition(cond) {
			results = append(results, StepResult{
				Name:    step.Name,
				Skipped: true,
				Passed:  true,
			})
			return results
		}
	}

	// Handle wait
	if step.Wait != "" {
		dur := parseDuration(step.Wait)
		time.Sleep(dur)
	}

	// If step has ordered statements (supports multi-run), execute them in order
	if len(step.Statements) > 0 {
		return e.executeStatementsOrdered(step, results)
	}

	// Legacy single-run path (backward compat for steps without Statements)
	if step.Run != nil {
		// Process let statements that DON'T reference last.* BEFORE run
		for _, l := range step.Lets {
			if !strings.HasPrefix(strings.TrimSpace(l.Value), "last.") {
				val := e.resolveLetValue(l.Value)
				e.Vars.Set(l.Name, val)
			}
		}

		result := e.executeRun(step.Run, step.Name)

		// Process let statements that reference last.* AFTER run
		for _, l := range step.Lets {
			if strings.HasPrefix(strings.TrimSpace(l.Value), "last.") {
				val := e.resolveLetValue(l.Value)
				e.Vars.Set(l.Name, val)
			}
		}

		// Execute log statements
		for _, msg := range step.Logs {
			resolved := e.interpolateWithLast(msg)
			fmt.Printf("  📋 log: %s\n", resolved)
		}

		// Execute write statements
		for _, w := range step.Writes {
			e.executeWrite(w)
		}

		results = append(results, result)
	} else {
		// No run — process lets and logs
		for _, l := range step.Lets {
			val := e.resolveLetValue(l.Value)
			e.Vars.Set(l.Name, val)
		}
		for _, msg := range step.Logs {
			resolved := e.interpolateWithLast(msg)
			fmt.Printf("  📋 log: %s\n", resolved)
		}
		for _, w := range step.Writes {
			e.executeWrite(w)
		}
	}

	// Handle retry
	if step.Retry != nil {
		result := e.executeRetry(step.Retry, step.Name)
		results = append(results, result)
	}

	// If no run and no retry, just record the step
	if step.Run == nil && step.Retry == nil && step.Wait == "" {
		results = append(results, StepResult{Name: step.Name, Passed: true})
	}

	return results
}

func (e *Engine) executeRun(run *ast.RunDecl, stepName string) StepResult {
	reqDef, ok := e.Requests[run.Name]
	if !ok {
		return StepResult{
			Name:   stepName,
			Error:  fmt.Sprintf("unknown request '%s'", run.Name),
			Passed: false,
		}
	}

	// Clone request for execution (apply args & overrides)
	req := e.prepareRequest(reqDef, run)

	result := e.ExecuteRequest(req)
	result.Name = stepName
	return result
}

func (e *Engine) prepareRequest(def *ast.RequestDecl, run *ast.RunDecl) *ast.RequestDecl {
	// Create a working copy
	req := *def

	// Apply run arguments as variables
	if len(run.Args) > 0 && len(def.Params) > 0 {
		for i, param := range def.Params {
			if i < len(run.Args) {
				val := run.Args[i]
				// If arg is a variable name, resolve it
				if resolved, ok := e.Vars.Get(val); ok {
					e.Vars.Set(param, resolved)
				} else {
					e.Vars.Set(param, val)
				}
			}
		}
	}

	// Apply overrides
	if run.Override != nil {
		if run.Override.Body != nil {
			if req.Body == nil {
				req.Body = run.Override.Body
			} else if run.Override.Body.Type == "schema" {
				// Schema override: replace body entirely or merge SetOverrides
				if req.Body.Type == "schema" {
					// Merge set overrides into existing schema body
					if req.Body.SetOverrides == nil {
						req.Body.SetOverrides = make(map[string]string)
					}
					for k, v := range run.Override.Body.SetOverrides {
						req.Body.SetOverrides[k] = v
					}
					// If override has different schema path, use it
					if run.Override.Body.SchemaPath != "" {
						req.Body.SchemaPath = run.Override.Body.SchemaPath
					}
				} else {
					// Replace non-schema body with schema body
					req.Body = run.Override.Body
				}
			} else {
				// Merge body fields (json/form)
				for _, f := range run.Override.Body.Fields {
					found := false
					for i, existing := range req.Body.Fields {
						if existing.Key == f.Key {
							req.Body.Fields[i].Value = f.Value
							found = true
							break
						}
					}
					if !found {
						req.Body.Fields = append(req.Body.Fields, f)
					}
				}
			}
		}
		if len(run.Override.Headers) > 0 {
			req.Headers = append(req.Headers, run.Override.Headers...)
		}
		if len(run.Override.Queries) > 0 {
			req.Queries = append(req.Queries, run.Override.Queries...)
		}
		if len(run.Override.Expects) > 0 {
			req.Expects = append(req.Expects, run.Override.Expects...)
		}
	}

	return &req
}

func (e *Engine) executeRetry(retry *ast.RetryDecl, stepName string) StepResult {
	interval := parseDuration(retry.Interval)
	var lastResult StepResult

	for i := 0; i < retry.Times; i++ {
		if i > 0 {
			time.Sleep(interval)
		}

		if retry.Run != nil {
			lastResult = e.executeRun(retry.Run, stepName)
			// TODO: check retry condition against response
			if lastResult.Passed {
				return lastResult
			}
		}
	}

	lastResult.Name = stepName
	if !lastResult.Passed {
		lastResult.Error = fmt.Sprintf("retry exhausted after %d attempts", retry.Times)
	}
	return lastResult
}

func (e *Engine) executeTeardown(td *ast.TeardownDecl) {
	if td.When != "" && !e.evalCondition(td.When) {
		return
	}
	if td.Run != nil {
		result := e.executeRun(td.Run, "teardown: "+td.Name)
		if !td.IgnoreFail && !result.Passed {
			// Log teardown failure but don't affect main results
		}
		_ = result
	}
	for _, step := range td.Steps {
		e.executeStep(&step)
	}
}

func (e *Engine) evalCondition(cond string) bool {
	// Simple condition: just a variable name → check if it exists and is non-empty
	cond = strings.TrimSpace(cond)

	// Handle "var == value" patterns
	if strings.Contains(cond, "==") {
		parts := strings.SplitN(cond, "==", 2)
		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])
		leftVal, _ := e.Vars.Get(left)
		rightVal := e.Vars.Interpolate(right)
		return leftVal == rightVal
	}

	if strings.Contains(cond, "!=") {
		parts := strings.SplitN(cond, "!=", 2)
		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])
		leftVal, _ := e.Vars.Get(left)
		rightVal := e.Vars.Interpolate(right)
		return leftVal != rightVal
	}

	// Simple: check variable exists and is non-empty
	// Could be multiple conditions with &&
	conditions := strings.Split(cond, "&&")
	for _, c := range conditions {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		val, exists := e.Vars.Get(c)
		if !exists || val == "" {
			return false
		}
	}
	return true
}

// executeStatementsOrdered executes step statements in declaration order.
// This supports multiple `run` commands with `let` extracting from each.
func (e *Engine) executeStatementsOrdered(step *ast.StepDecl, results []StepResult) []StepResult {
	var lastRunResult *StepResult

	for _, stmt := range step.Statements {
		switch stmt.Type {
		case "run":
			result := e.executeRun(stmt.Run, step.Name)
			lastRunResult = &result
			results = append(results, result)

		case "let":
			val := e.resolveLetValue(stmt.Let.Value)
			e.Vars.Set(stmt.Let.Name, val)

		case "log":
			resolved := e.interpolateWithLast(stmt.Log)
			fmt.Printf("  📋 log: %s\n", resolved)

		case "write":
			e.executeWrite(*stmt.Write)

		case "transform":
			e.executeTransform(stmt.Transform)

		case "expect":
			// Step-level expect evaluated against last response
			if e.LastResponse != nil && stmt.Expect != nil {
				ar := EvalExpect(stmt.Expect, e.LastResponse, e.Vars)
				if lastRunResult != nil {
					lastRunResult.Assertions = append(lastRunResult.Assertions, ar)
					if !ar.Passed && !ar.Soft {
						lastRunResult.Passed = false
					}
				}
			}
		}
	}

	// If no run was executed in statements, record a pass
	if lastRunResult == nil && len(results) == 0 {
		results = append(results, StepResult{Name: step.Name, Passed: true})
	}

	return results
}

func hasFailure(results []StepResult) bool {
	for _, r := range results {
		if !r.Passed && !r.Skipped {
			return true
		}
	}
	return false
}

// executeWrite handles `write <source> to "path"` statements.
func (e *Engine) executeWrite(w ast.WriteDecl) {
	if e.LastResponse == nil {
		fmt.Printf("  ⚠ write: no response available\n")
		return
	}

	// Resolve the path with variable interpolation
	outputPath := e.Vars.Interpolate(w.Path)
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(e.BaseDir, outputPath)
	}

	// Resolve source content
	var content []byte
	source := strings.TrimSpace(w.Source)

	switch {
	case source == "last.body":
		// Pretty-print if it's JSON
		content = prettyJSON(e.LastResponse.Body)

	case strings.HasPrefix(source, "last.json("):
		path := source[10 : len(source)-1]
		path = strings.Trim(path, "\"'")
		var body interface{}
		if err := json.Unmarshal(e.LastResponse.Body, &body); err == nil {
			val, found := evalJSONPath(body, path)
			if found {
				data, _ := json.MarshalIndent(val, "", "  ")
				content = data
			} else {
				fmt.Printf("  ⚠ write: path %q not found in response\n", path)
				return
			}
		} else {
			fmt.Printf("  ⚠ write: response is not valid JSON\n")
			return
		}

	case strings.HasPrefix(source, "last.header("):
		name := source[12 : len(source)-1]
		name = strings.Trim(name, "\"'")
		content = []byte(e.LastResponse.Headers.Get(name))

	case source == "last.status":
		content = []byte(fmt.Sprintf("%d", e.LastResponse.StatusCode))

	default:
		// Interpolate as variable/template
		content = []byte(e.Vars.Interpolate(source))
	}

	// Write to file
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("  ⚠ write: cannot create directory %s: %s\n", dir, err)
		return
	}

	var err error
	if w.Append {
		f, ferr := os.OpenFile(outputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if ferr != nil {
			err = ferr
		} else {
			_, err = f.Write(append(content, '\n'))
			f.Close()
		}
	} else {
		err = os.WriteFile(outputPath, content, 0644)
	}

	if err != nil {
		fmt.Printf("  ⚠ write: %s\n", err)
	} else {
		fmt.Printf("  💾 write: %s (%d bytes)\n", outputPath, len(content))
	}
}

// executeTransform executes a transform block: extracts array via JSONPath,
// maps each element to a new object using field mappings, and stores the result as a JSON string variable.
func (e *Engine) executeTransform(t *ast.TransformDecl) {
	if e.LastResponse == nil {
		fmt.Printf("  ⚠ transform %s: no response available (transform requires a preceding request)\n", t.Variable)
		return
	}

	// Parse response body
	var body interface{}
	if err := json.Unmarshal(e.LastResponse.Body, &body); err != nil {
		fmt.Printf("  ⚠ transform %s: response is not valid JSON: %s\n", t.Variable, err)
		return
	}

	// Evaluate JSONPath to extract source array
	result, found := evalJSONPath(body, t.JSONPath)
	if !found {
		fmt.Printf("  ⚠ transform %s: JSONPath '%s' not found in response body\n", t.Variable, t.JSONPath)
		return
	}

	// Verify result is an array
	sourceArray, ok := result.([]interface{})
	if !ok {
		fmt.Printf("  ⚠ transform %s: JSONPath '%s' resolved to %s, expected array\n", t.Variable, t.JSONPath, getJSONType(result))
		return
	}

	// Apply field mappings to each element
	var transformed []interface{}
	for _, elem := range sourceArray {
		elemMap, ok := elem.(map[string]interface{})
		if !ok {
			// If element is not an object, skip it
			continue
		}

		newObj := make(map[string]interface{})
		for _, mapping := range t.Mappings {
			val := e.resolveFieldMappingValue(mapping, elemMap)
			newObj[mapping.TargetName] = val
		}
		transformed = append(transformed, newObj)
	}

	// Serialize to JSON and store as variable
	data, err := json.Marshal(transformed)
	if err != nil {
		fmt.Printf("  ⚠ transform %s: failed to serialize result: %s\n", t.Variable, err)
		return
	}

	e.Vars.Set(t.Variable, string(data))
	if e.Verbose >= 1 {
		fmt.Printf("  🔄 transform %s: %d items mapped\n", t.Variable, len(transformed))
	}
}

// resolveFieldMappingValue resolves a single field mapping value for a given source element.
func (e *Engine) resolveFieldMappingValue(mapping ast.FieldMapping, elem map[string]interface{}) interface{} {
	switch mapping.ValueType {
	case "item_field":
		return resolveNestedField(elem, mapping.SourcePath)

	case "coercion":
		rawVal := resolveNestedField(elem, mapping.SourcePath)
		return applyCoercion(mapping.Coercion, rawVal, mapping.TargetName)

	case "static_number":
		// Try integer first, then float
		if n, err := strconv.ParseInt(mapping.StaticVal, 10, 64); err == nil {
			return n
		}
		if f, err := strconv.ParseFloat(mapping.StaticVal, 64); err == nil {
			return f
		}
		return 0

	case "static_string":
		return mapping.StaticVal

	case "static_bool":
		return mapping.StaticVal == "true"

	case "variable":
		// Interpolate {{variable}} references
		return e.Vars.Interpolate(mapping.StaticVal)

	default:
		return nil
	}
}

// resolveNestedField resolves a dot-separated path against a map.
// e.g. "address.city" resolves map["address"].(map)["city"]
func resolveNestedField(obj map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	var current interface{} = obj

	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		val, exists := m[part]
		if !exists {
			return nil
		}
		current = val
	}
	return current
}

// applyCoercion applies a type coercion function to a value.
func applyCoercion(fn string, val interface{}, fieldName string) interface{} {
	switch fn {
	case "number":
		switch v := val.(type) {
		case float64:
			return v
		case int64:
			return v
		case string:
			// Try integer first
			if n, err := strconv.ParseInt(v, 10, 64); err == nil {
				return n
			}
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f
			}
			fmt.Printf("  ⚠ transform: number() failed for field '%s': cannot parse '%s' as number\n", fieldName, v)
			return 0
		default:
			return 0
		}

	case "string":
		if val == nil {
			return ""
		}
		return fmt.Sprintf("%v", val)

	case "bool":
		switch v := val.(type) {
		case bool:
			return v
		case float64:
			return v != 0
		case int64:
			return v != 0
		case string:
			return v != ""
		case nil:
			return false
		default:
			return false
		}

	default:
		return val
	}
}

// prettyJSON attempts to pretty-print JSON, returns as-is if not valid JSON.
func prettyJSON(data []byte) []byte {
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err == nil {
		pretty, err := json.MarshalIndent(obj, "", "  ")
		if err == nil {
			return pretty
		}
	}
	return data
}

// buildSchemaBody loads a JSON Schema, generates payload, applies overrides.
func (e *Engine) buildSchemaBody(body *ast.BodyDecl) []byte {
	schemaPath := e.Vars.Interpolate(body.SchemaPath)
	// Resolve relative to project base
	if !filepath.IsAbs(schemaPath) {
		schemaPath = filepath.Join(e.BaseDir, schemaPath)
	}

	rawData, err := os.ReadFile(schemaPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ⚠ Schema/file error: cannot read %s: %s\n", schemaPath, err)
		return nil
	}

	// Determine if this is a JSON Schema or a raw JSON data template.
	// JSON Schema has "type" + "properties" at top level.
	// Raw JSON data template is any other valid JSON.
	var payload interface{}

	if isJSONSchema(rawData) {
		// Parse as JSON Schema → generate payload from defaults/examples
		s, err := schema.LoadSchema(schemaPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ⚠ Schema error: %s\n", err)
			return nil
		}
		payload = schema.GeneratePayload(s)
	} else {
		// Treat as raw JSON data template — load as-is
		if err := json.Unmarshal(rawData, &payload); err != nil {
			fmt.Fprintf(os.Stderr, "  ⚠ JSON file error: %s is not valid JSON: %s\n", schemaPath, err)
			return nil
		}
	}

	// Apply set overrides with variable interpolation
	if body.SetOverrides != nil && len(body.SetOverrides) > 0 {
		resolvedOverrides := make(map[string]string)
		for k, v := range body.SetOverrides {
			resolvedOverrides[k] = e.Vars.Interpolate(v)
		}
		payload = schema.ApplyOverrides(payload, resolvedOverrides)
	}

	// Interpolate any {{variables}} in string values throughout the payload
	payload = interpolatePayload(payload, e.Vars)

	data, err := json.Marshal(payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ⚠ Marshal error: %s\n", err)
		return nil
	}
	return data
}

// isJSONSchema checks if a JSON file is a JSON Schema (has "type" and "properties").
func isJSONSchema(data []byte) bool {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return false
	}
	_, hasType := raw["type"]
	_, hasProperties := raw["properties"]
	return hasType && hasProperties
}

// interpolatePayload recursively interpolates {{var}} in string values of a payload.
func interpolatePayload(data interface{}, vars *Variables) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, val := range v {
			result[key] = interpolatePayload(val, vars)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = interpolatePayload(val, vars)
		}
		return result
	case string:
		interpolated := vars.Interpolate(v)
		// If the interpolated result is a JSON array/object, parse and inject it
		trimmed := strings.TrimSpace(interpolated)
		if (strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) ||
			(strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) {
			var parsed interface{}
			if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
				return parsed
			}
		}
		return interpolated
	default:
		return data
	}
}

// interpolateWithLast extends variable interpolation to support {{last.json("$.path")}}
// and {{last.header("Name")}} inside strings (log messages, etc).
func (e *Engine) interpolateWithLast(s string) string {
	// First do standard interpolation (handles {{var}}, {{last.status}}, {{last.body}})
	result := e.Vars.Interpolate(s)

	// Then resolve any remaining {{last.json(...)}} or {{last.header(...)}} patterns
	re := regexp.MustCompile(`\{\{(last\.(json|header)\([^}]+\))\}\}`)
	result = re.ReplaceAllStringFunc(result, func(match string) string {
		expr := strings.TrimSpace(match[2 : len(match)-2])
		return e.resolveLetValue(expr)
	})

	return result
}

// resolveLetValue resolves a let value, handling special syntax:
// - last.json("$.path") → extract from last response body
// - last.header("Name") → extract from last response header
// - last.status → last response status code
// - regular string → interpolate variables
func (e *Engine) resolveLetValue(value string) string {
	value = strings.TrimSpace(value)

	// last.json("$.path")
	if strings.HasPrefix(value, "last.json(") && strings.HasSuffix(value, ")") {
		path := value[10 : len(value)-1]
		path = strings.Trim(path, "\"'")
		if e.LastResponse == nil {
			return ""
		}
		var body interface{}
		if err := json.Unmarshal(e.LastResponse.Body, &body); err != nil {
			return ""
		}
		result, found := evalJSONPath(body, path)
		if !found {
			return ""
		}
		return fmt.Sprintf("%v", result)
	}

	// last.header("Name")
	if strings.HasPrefix(value, "last.header(") && strings.HasSuffix(value, ")") {
		name := value[12 : len(value)-1]
		name = strings.Trim(name, "\"'")
		if e.LastResponse == nil {
			return ""
		}
		return e.LastResponse.Headers.Get(name)
	}

	// last.status
	if value == "last.status" {
		if e.LastResponse == nil {
			return ""
		}
		return fmt.Sprintf("%d", e.LastResponse.StatusCode)
	}

	// Regular value — interpolate
	return e.Vars.Interpolate(value)
}
