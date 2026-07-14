package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/testing-cli/apitest/internal/parser"
)

// LintOptions for the lint command.
type LintOptions struct {
	Path string
}

// RunLint validates FlowSpec files for syntax errors.
func RunLint(opts LintOptions) int {
	target := opts.Path
	if target == "" {
		target = "."
	}

	info, err := os.Stat(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s not found\n", target)
		return 2
	}

	var files []string
	if info.IsDir() {
		filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, ".flow") {
				files = append(files, path)
			}
			return nil
		})
	} else {
		files = []string{target}
	}

	totalErrors := 0
	totalWarnings := 0

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ Cannot read %s: %s\n", file, err)
			totalErrors++
			continue
		}

		p := parser.New(string(content), file)
		_ = p.Parse()

		errors := p.Errors()
		if len(errors) == 0 {
			fmt.Printf("  ✓ %s\n", file)
		} else {
			for _, e := range errors {
				fmt.Printf("  ✗ %s\n", e.Error())
				totalErrors++
			}
		}
	}

	fmt.Println()
	if totalErrors == 0 {
		fmt.Printf("✓ All %d files valid", len(files))
		if totalWarnings > 0 {
			fmt.Printf(" (%d warnings)", totalWarnings)
		}
		fmt.Println()
		return 0
	}

	fmt.Printf("✗ %d error(s) found in %d file(s)\n", totalErrors, len(files))
	return 1
}
