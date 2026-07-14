package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/testing-cli/apitest/internal/reporter"
	"github.com/testing-cli/apitest/internal/runtime"
)

// RunOptions holds options for the run command.
type RunOptions struct {
	Path      string
	Env       string
	Vars      map[string]string
	Tags      []string
	Verbose   int
	Quiet     bool
	FailFast  bool
	NoColor   bool
	Timeout   time.Duration
	ReportDir string
	Reporters []string
}

// RunTest executes test files.
func RunTest(opts RunOptions) int {
	baseDir := findProjectRoot()
	if baseDir == "" {
		baseDir = "."
	}

	engine := runtime.NewEngine(baseDir, opts.Timeout, opts.Verbose, opts.FailFast)

	// Load environment
	if opts.Env != "" {
		envPath := filepath.Join(baseDir, "env", opts.Env+".flow")
		if _, err := os.Stat(envPath); err == nil {
			if err := engine.LoadEnvFile(envPath, opts.Env); err != nil {
				fmt.Fprintf(os.Stderr, "Error loading env: %s\n", err)
				return 2
			}
		}
	}

	// Load .env file if exists
	loadDotEnv(baseDir)

	// Apply CLI variable overrides
	for k, v := range opts.Vars {
		engine.Vars.Set(k, v)
	}

	// Load shared files (auth, fragments)
	sharedDir := filepath.Join(baseDir, "shared")
	if info, err := os.Stat(sharedDir); err == nil && info.IsDir() {
		loadFlowDir(engine, sharedDir)
	}

	// Determine target path
	target := opts.Path
	if target == "" {
		target = "."
	}

	info, err := os.Stat(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s not found\n", target)
		return 2
	}

	var allResults []runtime.StepResult
	var flowName string

	if info.IsDir() {
		// Run all .flow files in directory
		results, name := runDirectory(engine, target, opts.Tags)
		allResults = results
		flowName = name
	} else {
		// Run single file
		results, name := runFile(engine, target, opts.Tags)
		allResults = results
		flowName = name
	}

	// Report results
	consoleReporter := &reporter.ConsoleReporter{
		NoColor: opts.NoColor,
		Verbose: opts.Verbose,
		Quiet:   opts.Quiet,
	}
	consoleReporter.ReportResults(flowName, allResults)

	// Generate file reports
	for _, rep := range opts.Reporters {
		switch rep {
		case "json":
			reportDir := opts.ReportDir
			if reportDir == "" {
				reportDir = filepath.Join(baseDir, "reports")
			}
			if err := reporter.WriteJSONReport(allResults, reportDir); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing JSON report: %s\n", err)
			}
		case "junit":
			reportDir := opts.ReportDir
			if reportDir == "" {
				reportDir = filepath.Join(baseDir, "reports")
			}
			if err := reporter.WriteJUnitReport(allResults, flowName, reportDir); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing JUnit report: %s\n", err)
			}
		}
	}

	// Determine exit code
	for _, r := range allResults {
		if !r.Passed && !r.Skipped {
			return 1
		}
	}
	return 0
}

func runFile(engine *runtime.Engine, path string, tags []string) ([]runtime.StepResult, string) {
	file, err := engine.LoadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		return []runtime.StepResult{{Name: path, Error: err.Error()}}, path
	}

	var allResults []runtime.StepResult
	flowName := ""

	// Execute requests in file
	for i := range file.Requests {
		req := &file.Requests[i]
		if !matchesTags(req.Tags, tags) {
			continue
		}
		result := engine.ExecuteRequest(req)
		allResults = append(allResults, result)
		if flowName == "" {
			flowName = req.Name
		}
	}

	// Execute flows in file
	for i := range file.Flows {
		flow := &file.Flows[i]
		if !matchesTags(flow.Tags, tags) {
			continue
		}
		flowName = flow.Name
		results := engine.ExecuteFlow(flow)
		allResults = append(allResults, results...)
	}

	return allResults, flowName
}

func runDirectory(engine *runtime.Engine, dir string, tags []string) ([]runtime.StepResult, string) {
	var allResults []runtime.StepResult
	flowName := filepath.Base(dir)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".flow") {
			return nil
		}
		results, _ := runFile(engine, path, tags)
		allResults = append(allResults, results...)
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error walking directory: %s\n", err)
	}

	return allResults, flowName
}

func matchesTags(itemTags []string, filterTags []string) bool {
	if len(filterTags) == 0 {
		return true
	}
	for _, ft := range filterTags {
		for _, it := range itemTags {
			if it == ft {
				return true
			}
		}
	}
	return false
}

func findProjectRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "apitest.flow")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func loadDotEnv(baseDir string) {
	envFile := filepath.Join(baseDir, ".env")
	data, err := os.ReadFile(envFile)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, "\"'")
			os.Setenv(key, val)
		}
	}
}

func loadFlowDir(engine *runtime.Engine, dir string) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".flow") {
			return nil
		}
		engine.LoadFile(path)
		return nil
	})
}
