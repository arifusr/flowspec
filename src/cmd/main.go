package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/testing-cli/apitest/internal/cli"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	command := os.Args[1]

	switch command {
	case "init":
		dir := ""
		if len(os.Args) > 2 {
			dir = os.Args[2]
		}
		if err := cli.RunInit(dir); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(2)
		}

	case "run":
		opts := parseRunOptions()
		exitCode := cli.RunTest(opts)
		os.Exit(exitCode)

	case "dsl":
		if len(os.Args) < 3 {
			fmt.Println("Usage: apitest dsl <lint|show|fmt>")
			os.Exit(2)
		}
		subCmd := os.Args[2]
		switch subCmd {
		case "lint":
			path := "."
			if len(os.Args) > 3 {
				path = os.Args[3]
			}
			exitCode := cli.RunLint(cli.LintOptions{Path: path})
			os.Exit(exitCode)
		case "show":
			path := ""
			env := ""
			args := os.Args[3:]
			for i := 0; i < len(args); i++ {
				if args[i] == "--env" && i+1 < len(args) {
					i++
					env = args[i]
				} else if !strings.HasPrefix(args[i], "-") {
					path = args[i]
				}
			}
			if path == "" {
				fmt.Fprintln(os.Stderr, "Usage: apitest dsl show <file> [--env <name>]")
				os.Exit(2)
			}
			exitCode := cli.RunShow(cli.ShowOptions{Path: path, Env: env})
			os.Exit(exitCode)
		case "fmt":
			fmt.Println("dsl fmt: not yet implemented")
		default:
			fmt.Fprintf(os.Stderr, "Unknown dsl subcommand: %s\n", subCmd)
			os.Exit(2)
		}

	case "--version", "version":
		fmt.Printf("apitest v%s\n", version)
		fmt.Println("Documentation: https://github.com/arifusr/flowspec")

	case "--help", "help", "-h":
		if len(os.Args) > 2 {
			printCommandHelp(os.Args[2])
		} else {
			printUsage()
		}

	case "import":
		if len(os.Args) < 3 {
			fmt.Println("Usage: apitest import <curl|openapi|postman> ...")
			os.Exit(2)
		}
		subCmd := os.Args[2]
		switch subCmd {
		case "curl":
			curlCmd := ""
			output := ""
			file := ""
			outputDir := ""
			args := os.Args[3:]
			for i := 0; i < len(args); i++ {
				switch args[i] {
				case "--output":
					i++
					if i < len(args) {
						output = args[i]
					}
				case "--file", "-f":
					i++
					if i < len(args) {
						file = args[i]
					}
				case "--output-dir":
					i++
					if i < len(args) {
						outputDir = args[i]
					}
				default:
					if !strings.HasPrefix(args[i], "-") && curlCmd == "" {
						curlCmd = args[i]
					}
				}
			}
			if curlCmd == "" && file == "" {
				fmt.Fprintln(os.Stderr, `Usage:
  apitest import curl '<curl command>' [--output <file>]
  apitest import curl --file <path> [--output-dir <dir>]`)
				os.Exit(2)
			}
			if err := cli.RunImportCurl(cli.ImportCurlOptions{
				CurlCommand: curlCmd,
				Output:      output,
				File:        file,
				OutputDir:   outputDir,
			}); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				os.Exit(1)
			}
		default:
			fmt.Fprintf(os.Stderr, "Import type '%s' not yet implemented\n", subCmd)
			os.Exit(2)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		fmt.Fprintln(os.Stderr, "Run 'apitest help' for usage.")
		os.Exit(2)
	}
}

func parseRunOptions() cli.RunOptions {
	opts := cli.RunOptions{
		Timeout: 30 * time.Second,
		Vars:    make(map[string]string),
	}

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--env" && i+1 < len(args):
			i++
			opts.Env = args[i]
		case arg == "--var" && i+1 < len(args):
			i++
			parts := strings.SplitN(args[i], "=", 2)
			if len(parts) == 2 {
				opts.Vars[parts[0]] = parts[1]
			}
		case arg == "--tags" && i+1 < len(args):
			i++
			opts.Tags = strings.Split(args[i], ",")
		case arg == "-v":
			opts.Verbose = 1
		case arg == "-vv":
			opts.Verbose = 2
		case arg == "-vvv":
			opts.Verbose = 3
		case arg == "--quiet" || arg == "-q":
			opts.Quiet = true
		case arg == "--fail-fast":
			opts.FailFast = true
		case arg == "--no-color":
			opts.NoColor = true
		case arg == "--timeout" && i+1 < len(args):
			i++
			if d, err := time.ParseDuration(args[i]); err == nil {
				opts.Timeout = d
			}
		case arg == "--report" && i+1 < len(args):
			i++
			opts.Reporters = strings.Split(args[i], ",")
		case arg == "--output" && i+1 < len(args):
			i++
			opts.ReportDir = args[i]
		case !strings.HasPrefix(arg, "-"):
			opts.Path = arg
		}
	}
	return opts
}

func printUsage() {
	fmt.Printf(`apitest v%s — CLI API Testing Tool with FlowSpec DSL

Usage:
  apitest <command> [options]

Commands:
  init                   Create a new FlowSpec project
  run <path>             Run test files (.flow)
  dsl lint <path>        Validate FlowSpec syntax
  dsl show <file>        Preview resolved request (dry run)
  dsl fmt <path>         Format FlowSpec files
  import curl            Import from cURL command or file
  help <command>         Show help for a command
  version                Show version

Run Options:
  --env <name>           Select environment (dev, staging, prod)
  --var key=value        Override variable
  --tags tag1,tag2       Filter tests by tag
  --fail-fast            Stop on first failure
  --timeout <duration>   Global timeout (default: 30s)
  --report json,junit    Generate report files
  --output <dir>         Report output directory
  -v / -vv / -vvv       Verbose output
  -q / --quiet           Minimal output (CI mode)
  --no-color             Disable colored output

Examples:
  apitest init
  apitest run requests/users/list-users.flow --env dev
  apitest run flows/user-crud.flow --env staging -v
  apitest run flows/ --env staging --tags smoke --report json
  apitest import curl 'curl -X GET https://api.example.com/users'
  apitest import curl --file curls.txt --output-dir requests/
  apitest dsl lint .

Documentation: https://github.com/arifusr/flowspec
`, version)
}

func printCommandHelp(cmd string) {
	switch cmd {
	case "run":
		fmt.Println(`apitest run — Execute test files

Usage:
  apitest run <file-or-directory> [options]

Examples:
  apitest run requests/users/list-users.flow --env dev
  apitest run flows/ --env staging --tags smoke
  apitest run flows/user-crud.flow --env dev -vv --fail-fast`)
	case "init":
		fmt.Println(`apitest init — Create a new FlowSpec project

Usage:
  apitest init [directory]

Creates the standard project structure with env/, requests/, flows/, etc.`)
	case "dsl":
		fmt.Println(`apitest dsl — DSL utilities

Subcommands:
  lint <path>    Validate FlowSpec syntax
  show <file>    Preview resolved request
  fmt <path>     Format/prettify FlowSpec files`)
	case "import":
		fmt.Println(`apitest import — Import from external formats

Subcommands:
  curl    Import from cURL command or file

Usage:
  apitest import curl '<curl command>' [--output <file>]
  apitest import curl --file <path> [--output-dir <dir>]

Options:
  --output <file>      Output .flow file path (single command)
  --file, -f <path>    Read curl command(s) from a file
  --output-dir <dir>   Output directory (default: requests/)

File Format:
  The curl file can contain one or multiple curl commands.
  Commands can span multiple lines using backslash (\) continuation.
  Commands are separated by blank lines.
  Lines starting with # or // are treated as comments.

Examples:
  apitest import curl 'curl -X GET https://api.example.com/users'
  apitest import curl --file api-calls.txt
  apitest import curl --file curls.txt --output-dir requests/users/`)
	default:
		fmt.Printf("No help available for '%s'\n", cmd)
	}
}
