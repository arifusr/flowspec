package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/testing-cli/apitest/internal/runtime"
)

// ShowOptions for the dsl show command.
type ShowOptions struct {
	Path string
	Env  string
}

// RunShow displays the resolved request/flow (dry run preview).
func RunShow(opts ShowOptions) int {
	baseDir := findProjectRoot()
	if baseDir == "" {
		baseDir = "."
	}

	engine := runtime.NewEngine(baseDir, 30*time.Second, 0, false)

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

	loadDotEnv(baseDir)

	// Load shared
	sharedDir := filepath.Join(baseDir, "shared")
	if info, err := os.Stat(sharedDir); err == nil && info.IsDir() {
		loadFlowDir(engine, sharedDir)
	}

	file, err := engine.LoadFile(opts.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		return 2
	}

	// Show requests
	for _, req := range file.Requests {
		fmt.Printf("Request: %s\n", req.Name)
		if len(req.Params) > 0 {
			fmt.Printf("  Params: %v\n", req.Params)
		}
		resolvedURL := engine.Vars.Interpolate(req.URL)
		fmt.Printf("  %s %s\n", req.Method, resolvedURL)

		if req.UseAuth != "" {
			fmt.Printf("  Auth: %s\n", req.UseAuth)
		}
		for _, h := range req.Headers {
			val := engine.Vars.Interpolate(h.Value)
			fmt.Printf("  Header: %s = %s\n", h.Key, val)
		}
		if req.Body != nil {
			fmt.Printf("  Body (%s):\n", req.Body.Type)
			for _, f := range req.Body.Fields {
				val := engine.Vars.Interpolate(f.Value)
				fmt.Printf("    %s: %s\n", f.Key, val)
			}
		}
		fmt.Printf("  Expects: %d assertions\n", len(req.Expects))
		fmt.Printf("  Extracts: %d variables\n", len(req.Extracts))
		fmt.Println()
	}

	// Show flows
	for _, flow := range file.Flows {
		fmt.Printf("Flow: %s", flow.Name)
		if flow.EnvTag != "" {
			fmt.Printf(" (env: %s)", flow.EnvTag)
		}
		fmt.Println()
		if flow.Description != "" {
			fmt.Printf("  Description: %s\n", flow.Description)
		}
		if len(flow.Tags) > 0 {
			fmt.Printf("  Tags: %v\n", flow.Tags)
		}
		fmt.Println("  Variables:")
		for _, l := range flow.Lets {
			val := engine.Vars.Interpolate(l.Value)
			fmt.Printf("    %s = %s\n", l.Name, val)
		}
		fmt.Println("  Steps:")
		for i, step := range flow.Steps {
			when := ""
			if step.When != "" {
				when = fmt.Sprintf(" [when %s]", step.When)
			}
			runInfo := ""
			if step.Run != nil {
				runInfo = fmt.Sprintf("→ %s", step.Run.Name)
				if len(step.Run.Args) > 0 {
					runInfo += fmt.Sprintf("(%v)", step.Run.Args)
				}
			}
			fmt.Printf("    %d. %-25s %s%s\n", i+1, step.Name, runInfo, when)
		}
		if flow.Teardown != nil {
			fmt.Printf("  Teardown: %s\n", flow.Teardown.Name)
		}
		fmt.Println()
	}

	return 0
}
