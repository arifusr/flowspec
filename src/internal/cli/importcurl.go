package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ImportCurlOptions for import curl command.
type ImportCurlOptions struct {
	CurlCommand string
	Output      string
	File        string // path to file containing curl command(s)
	OutputDir   string // output directory when importing from file with multiple commands
}

// RunImportCurl converts a cURL command to a .flow file.
// Supports both inline command string and reading from a file.
func RunImportCurl(opts ImportCurlOptions) error {
	// If --file is provided, read curl commands from file
	if opts.File != "" {
		return importCurlFromFile(opts)
	}

	// Inline curl command
	if opts.CurlCommand == "" {
		return fmt.Errorf("provide a curl command or use --file <path>")
	}

	flowContent, method, url, err := parseCurlToFlow(opts.CurlCommand)
	if err != nil {
		return err
	}

	// Determine output path
	output := opts.Output
	if output == "" {
		name := generateRequestName(url, method)
		output = fmt.Sprintf("requests/%s.flow", toKebabCase(name))
	}

	return writeFlowFile(output, flowContent, method, url)
}

// importCurlFromFile reads one or more curl commands from a file.
// Each command can span multiple lines using backslash continuation.
// Commands are separated by blank lines or lines starting with "curl".
func importCurlFromFile(opts ImportCurlOptions) error {
	data, err := os.ReadFile(opts.File)
	if err != nil {
		return fmt.Errorf("cannot read file %s: %w", opts.File, err)
	}

	commands := splitCurlCommands(string(data))
	if len(commands) == 0 {
		return fmt.Errorf("no curl commands found in %s", opts.File)
	}

	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = "requests"
	}

	fmt.Printf("Found %d curl command(s) in %s\n\n", len(commands), opts.File)

	for i, cmd := range commands {
		flowContent, method, url, err := parseCurlToFlow(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ Skipping command %d: %s\n", i+1, err)
			continue
		}

		// Generate output filename
		name := generateRequestName(url, method)
		output := opts.Output
		if output == "" || len(commands) > 1 {
			output = filepath.Join(outputDir, toKebabCase(name)+".flow")
		}

		if err := writeFlowFile(output, flowContent, method, url); err != nil {
			fmt.Fprintf(os.Stderr, "⚠ Error writing %s: %s\n", output, err)
			continue
		}
	}

	return nil
}

// splitCurlCommands splits a file content into individual curl commands.
// Handles multi-line commands with backslash continuation and detects
// command boundaries by "curl " prefix or blank line separation.
func splitCurlCommands(content string) []string {
	var commands []string
	var current strings.Builder

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip comments and empty lines
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			// If we have accumulated content, a comment line doesn't break it
			continue
		}

		// If line is empty and we have a command, finalize it
		if trimmed == "" {
			if current.Len() > 0 {
				commands = append(commands, current.String())
				current.Reset()
			}
			continue
		}

		// If line starts with "curl" and we already have content, start new command
		if strings.HasPrefix(trimmed, "curl") && current.Len() > 0 {
			commands = append(commands, current.String())
			current.Reset()
		}

		// Handle backslash continuation
		if strings.HasSuffix(trimmed, "\\") {
			trimmed = strings.TrimSuffix(trimmed, "\\")
			current.WriteString(trimmed)
			current.WriteString(" ")
		} else {
			current.WriteString(trimmed)
			current.WriteString(" ")
		}
	}

	// Don't forget the last command
	if current.Len() > 0 {
		commands = append(commands, strings.TrimSpace(current.String()))
	}

	return commands
}

// parseCurlToFlow converts a single curl command string to .flow content.
func parseCurlToFlow(cmd string) (flowContent, method, url string, err error) {
	cmd = strings.TrimSpace(cmd)
	if strings.HasPrefix(cmd, "curl ") {
		cmd = cmd[5:]
	} else if strings.HasPrefix(cmd, "curl") {
		cmd = cmd[4:]
	}

	method = "GET"
	url = ""
	var headers []string
	var dataRaw string

	parts := tokenizeCurl(cmd)

	for i := 0; i < len(parts); i++ {
		part := parts[i]
		switch {
		case part == "-X" || part == "--request":
			i++
			if i < len(parts) {
				method = strings.ToUpper(parts[i])
			}
		case part == "-H" || part == "--header":
			i++
			if i < len(parts) {
				headers = append(headers, parts[i])
			}
		case part == "-d" || part == "--data" || part == "--data-raw" || part == "--data-binary":
			i++
			if i < len(parts) {
				dataRaw = parts[i]
				if method == "GET" {
					method = "POST"
				}
			}
		case part == "-u" || part == "--user":
			i++
			if i < len(parts) {
				// Basic auth: user:password → add Authorization header
				headers = append(headers, "Authorization: Basic {{$base64("+parts[i]+")}}")
			}
		case part == "-L" || part == "--location" || part == "-k" || part == "--insecure" ||
			part == "-s" || part == "--silent" || part == "-S" || part == "--show-error" ||
			part == "-i" || part == "--include" || part == "-v" || part == "--verbose" ||
			part == "--compressed":
			// Known flags without arguments, skip
			continue
		case part == "-o" || part == "--output" || part == "--connect-timeout" ||
			part == "--max-time" || part == "-A" || part == "--user-agent":
			// Known flags with arguments, skip both
			i++
		case !strings.HasPrefix(part, "-"):
			url = strings.Trim(part, "'\"")
		}
	}

	if url == "" {
		return "", "", "", fmt.Errorf("no URL found in curl command")
	}

	requestName := generateRequestName(url, method)

	// Build .flow content
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("request %s {\n", requestName))
	sb.WriteString(fmt.Sprintf("  %s \"%s\"\n", method, url))

	if len(headers) > 0 {
		sb.WriteString("\n")
		for _, h := range headers {
			hParts := strings.SplitN(h, ":", 2)
			if len(hParts) == 2 {
				key := strings.TrimSpace(hParts[0])
				val := strings.TrimSpace(hParts[1])
				sb.WriteString(fmt.Sprintf("  header %s = \"%s\"\n", key, val))
			}
		}
	}

	if dataRaw != "" {
		sb.WriteString("\n  body json {\n")
		dataRaw = strings.TrimSpace(dataRaw)
		if strings.HasPrefix(dataRaw, "{") {
			inner := strings.TrimPrefix(dataRaw, "{")
			inner = strings.TrimSuffix(inner, "}")
			pairs := parseSimpleJSON(inner)
			for _, p := range pairs {
				sb.WriteString(fmt.Sprintf("    %s: \"%s\"\n", p[0], p[1]))
			}
		} else {
			sb.WriteString(fmt.Sprintf("    raw: \"%s\"\n", dataRaw))
		}
		sb.WriteString("  }\n")
	}

	sb.WriteString("\n")
	if method == "POST" {
		sb.WriteString("  expect status 201\n")
	} else if method == "DELETE" {
		sb.WriteString("  expect status 204\n")
	} else {
		sb.WriteString("  expect status 200\n")
	}
	sb.WriteString("}\n")

	return sb.String(), method, url, nil
}

func writeFlowFile(output, content, method, url string) error {
	dir := filepath.Dir(output)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(output, []byte(content), 0644); err != nil {
		return err
	}

	fmt.Printf("✓ Imported to %s\n", output)
	fmt.Printf("  %s %s\n", method, url)
	return nil
}

func tokenizeCurl(cmd string) []string {
	var tokens []string
	var current strings.Builder
	inQuote := byte(0)
	escaped := false

	for i := 0; i < len(cmd); i++ {
		ch := cmd[i]

		if escaped {
			current.WriteByte(ch)
			escaped = false
			continue
		}

		if ch == '\\' && inQuote != '\'' {
			// In double quotes or no quotes, backslash escapes next char
			if i+1 < len(cmd) {
				next := cmd[i+1]
				if next == '\n' {
					i++ // skip line continuation
					continue
				}
				if inQuote == '"' || inQuote == 0 {
					escaped = true
					continue
				}
			}
			current.WriteByte(ch)
			continue
		}

		switch {
		case ch == '\'' && inQuote == 0:
			inQuote = '\''
		case ch == '\'' && inQuote == '\'':
			inQuote = 0
		case ch == '"' && inQuote == 0:
			inQuote = '"'
		case ch == '"' && inQuote == '"':
			inQuote = 0
		case (ch == ' ' || ch == '\t' || ch == '\n') && inQuote == 0:
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

func generateRequestName(url, method string) string {
	// Extract last path segment
	parts := strings.Split(strings.TrimRight(url, "/"), "/")
	last := parts[len(parts)-1]
	// Remove query params
	if idx := strings.Index(last, "?"); idx != -1 {
		last = last[:idx]
	}
	// Clean non-alpha chars
	re := regexp.MustCompile(`[^a-zA-Z0-9]`)
	last = re.ReplaceAllString(last, "")

	if last == "" {
		last = "Request"
	}

	// Capitalize
	last = strings.ToUpper(last[:1]) + last[1:]

	prefix := ""
	switch method {
	case "GET":
		prefix = "Get"
	case "POST":
		prefix = "Create"
	case "PUT", "PATCH":
		prefix = "Update"
	case "DELETE":
		prefix = "Delete"
	}

	if !strings.HasPrefix(last, prefix) {
		return prefix + last
	}
	return last
}

func toKebabCase(s string) string {
	// PascalCase to kebab-case
	var result strings.Builder
	for i, ch := range s {
		if ch >= 'A' && ch <= 'Z' {
			if i > 0 {
				result.WriteByte('-')
			}
			result.WriteRune(ch + 32) // lowercase
		} else {
			result.WriteRune(ch)
		}
	}
	return result.String()
}

func parseSimpleJSON(s string) [][2]string {
	var pairs [][2]string
	s = strings.TrimSpace(s)
	// Simple key-value extraction (not full JSON parser)
	parts := strings.Split(s, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		kv := strings.SplitN(p, ":", 2)
		if len(kv) == 2 {
			key := strings.Trim(strings.TrimSpace(kv[0]), "\"'")
			val := strings.Trim(strings.TrimSpace(kv[1]), "\"'")
			pairs = append(pairs, [2]string{key, val})
		}
	}
	return pairs
}
