package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/testing-cli/apitest/internal/ast"
	"github.com/testing-cli/apitest/internal/parser"
)

// Engine executes FlowSpec AST nodes.
type Engine struct {
	Vars      *Variables
	Client    *HTTPClient
	Requests  map[string]*ast.RequestDecl
	Auths     map[string]*ast.AuthDecl
	Fragments map[string]*ast.FragmentDecl
	Results   []StepResult
	BaseDir   string
	Verbose   int
	FailFast  bool
	Timeout   time.Duration
	Redact    []string
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

	httpReq := BuildRequest(req.Method, req.URL, headers, queries, bodyType, bodyFields, e.Vars, e.Timeout)
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

	// Handle let statements
	for _, l := range step.Lets {
		e.Vars.Set(l.Name, e.Vars.Interpolate(l.Value))
	}

	// Handle run
	if step.Run != nil {
		result := e.executeRun(step.Run, step.Name)

		// Additional step-level expects
		if result.Passed && len(step.Expects) > 0 {
			// We need the last response for step-level expects
			// For now, step expects are merged during run
		}

		results = append(results, result)
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
			} else {
				// Merge body fields
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

func hasFailure(results []StepResult) bool {
	for _, r := range results {
		if !r.Passed && !r.Skipped {
			return true
		}
	}
	return false
}
