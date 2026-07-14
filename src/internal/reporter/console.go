package reporter

import (
	"fmt"
	"strings"
	"time"

	"github.com/testing-cli/apitest/internal/runtime"
)

// Colors for terminal output
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
)

// ConsoleReporter outputs results to terminal.
type ConsoleReporter struct {
	NoColor bool
	Verbose int
	Quiet   bool
}

// ReportResults prints step results to console.
func (r *ConsoleReporter) ReportResults(flowName string, results []runtime.StepResult) {
	if r.Quiet {
		r.reportQuiet(flowName, results)
		return
	}

	passed, failed, skipped := countResults(results)

	if flowName != "" {
		fmt.Printf("\nScenario: %s\n", flowName)
		fmt.Println(strings.Repeat("─", 50))
	}

	for i, res := range results {
		r.printStepResult(i+1, res)
	}

	fmt.Println()
	totalDuration := sumDuration(results)
	r.printSummary(passed, failed, skipped, totalDuration)
}

func (r *ConsoleReporter) printStepResult(num int, res runtime.StepResult) {
	if res.Skipped {
		fmt.Printf("  %s⊘ Step %d: %s%s    %sskipped%s\n",
			r.color(colorYellow), num, res.Name, r.color(colorReset),
			r.color(colorGray), r.color(colorReset))
		return
	}

	icon := r.color(colorGreen) + "✓" + r.color(colorReset)
	if !res.Passed {
		icon = r.color(colorRed) + "✗" + r.color(colorReset)
	}

	dur := formatDuration(res.Duration)
	status := ""
	if res.StatusCode > 0 {
		status = fmt.Sprintf("%d", res.StatusCode)
	}

	fmt.Printf("  %s Step %d: %-30s %s%s  %s%s%s\n",
		icon, num, res.Name, status, "",
		r.color(colorGray), dur, r.color(colorReset))

	if res.Error != "" {
		fmt.Printf("    %sError: %s%s\n", r.color(colorRed), res.Error, r.color(colorReset))
	}

	if r.Verbose > 0 && res.Method != "" {
		fmt.Printf("    %s%s %s%s\n",
			r.color(colorGray), res.Method, res.URL, r.color(colorReset))
	}

	// Print failed assertions
	for _, a := range res.Assertions {
		if !a.Passed {
			fmt.Printf("    %sAssertion failed:%s\n", r.color(colorRed), r.color(colorReset))
			fmt.Printf("      %s\n", a.Message)
			if a.Expected != "" {
				fmt.Printf("      Expected: %s\n", a.Expected)
				fmt.Printf("      Actual:   %s\n", a.Actual)
			}
		} else if r.Verbose >= 2 {
			fmt.Printf("    %s✓ %s%s\n", r.color(colorGreen), a.Message, r.color(colorReset))
		}
	}
}

func (r *ConsoleReporter) printSummary(passed, failed, skipped int, duration time.Duration) {
	total := passed + failed + skipped
	if failed == 0 {
		fmt.Printf("%sSummary: %d passed%s",
			r.color(colorGreen), passed, r.color(colorReset))
	} else {
		fmt.Printf("%sSummary: %d passed, %d failed%s",
			r.color(colorRed), passed, failed, r.color(colorReset))
	}
	if skipped > 0 {
		fmt.Printf(", %d skipped", skipped)
	}
	fmt.Printf(" (%d total, %s)\n", total, formatDuration(duration))
}

func (r *ConsoleReporter) reportQuiet(flowName string, results []runtime.StepResult) {
	passed, failed, _ := countResults(results)
	duration := sumDuration(results)
	if failed == 0 {
		fmt.Printf("%s✓%s %s (%d passed, %s)\n",
			r.color(colorGreen), r.color(colorReset), flowName, passed, formatDuration(duration))
	} else {
		fmt.Printf("%s✗%s %s (%d passed, %d failed, %s)\n",
			r.color(colorRed), r.color(colorReset), flowName, passed, failed, formatDuration(duration))
	}
}

func (r *ConsoleReporter) color(c string) string {
	if r.NoColor {
		return ""
	}
	return c
}

func countResults(results []runtime.StepResult) (passed, failed, skipped int) {
	for _, res := range results {
		if res.Skipped {
			skipped++
		} else if res.Passed {
			passed++
		} else {
			failed++
		}
	}
	return
}

func sumDuration(results []runtime.StepResult) time.Duration {
	var total time.Duration
	for _, res := range results {
		total += res.Duration
	}
	return total
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
