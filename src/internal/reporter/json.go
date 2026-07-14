package reporter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/testing-cli/apitest/internal/runtime"
)

// JSONReport is the structure of a JSON test report.
type JSONReport struct {
	Timestamp string        `json:"timestamp"`
	Duration  string        `json:"duration"`
	Summary   ReportSummary `json:"summary"`
	Results   []JSONResult  `json:"results"`
}

// ReportSummary contains aggregate results.
type ReportSummary struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

// JSONResult represents one step in the report.
type JSONResult struct {
	Name       string          `json:"name"`
	Request    string          `json:"request,omitempty"`
	Method     string          `json:"method,omitempty"`
	URL        string          `json:"url,omitempty"`
	StatusCode int             `json:"status_code,omitempty"`
	Duration   string          `json:"duration"`
	Passed     bool            `json:"passed"`
	Skipped    bool            `json:"skipped,omitempty"`
	Error      string          `json:"error,omitempty"`
	Assertions []JSONAssertion `json:"assertions,omitempty"`
}

// JSONAssertion is one assertion in the report.
type JSONAssertion struct {
	Passed   bool   `json:"passed"`
	Message  string `json:"message"`
	Expected string `json:"expected,omitempty"`
	Actual   string `json:"actual,omitempty"`
}

// WriteJSONReport writes a JSON report file.
func WriteJSONReport(results []runtime.StepResult, outputDir string) error {
	passed, failed, skipped := countResults(results)
	totalDuration := sumDuration(results)

	report := JSONReport{
		Timestamp: time.Now().Format(time.RFC3339),
		Duration:  totalDuration.String(),
		Summary: ReportSummary{
			Total:   passed + failed + skipped,
			Passed:  passed,
			Failed:  failed,
			Skipped: skipped,
		},
	}

	for _, r := range results {
		jr := JSONResult{
			Name:       r.Name,
			Request:    r.RequestName,
			Method:     r.Method,
			URL:        r.URL,
			StatusCode: r.StatusCode,
			Duration:   r.Duration.String(),
			Passed:     r.Passed,
			Skipped:    r.Skipped,
			Error:      r.Error,
		}
		for _, a := range r.Assertions {
			jr.Assertions = append(jr.Assertions, JSONAssertion{
				Passed:   a.Passed,
				Message:  a.Message,
				Expected: a.Expected,
				Actual:   a.Actual,
			})
		}
		report.Results = append(report.Results, jr)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	filename := filepath.Join(outputDir, "report-"+time.Now().Format("2006-01-02T15-04-05")+".json")
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}
