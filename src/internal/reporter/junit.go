package reporter

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/testing-cli/apitest/internal/runtime"
)

// JUnit XML structures
type junitTestSuites struct {
	XMLName xml.Name         `xml:"testsuites"`
	Suites  []junitTestSuite `xml:"testsuite"`
}

type junitTestSuite struct {
	XMLName  xml.Name        `xml:"testsuite"`
	Name     string          `xml:"name,attr"`
	Tests    int             `xml:"tests,attr"`
	Failures int             `xml:"failures,attr"`
	Skipped  int             `xml:"skipped,attr"`
	Time     string          `xml:"time,attr"`
	Cases    []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	XMLName   xml.Name      `xml:"testcase"`
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
	Skipped   *junitSkipped `xml:"skipped,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

type junitSkipped struct {
	Message string `xml:"message,attr,omitempty"`
}

// WriteJUnitReport writes a JUnit XML report file.
func WriteJUnitReport(results []runtime.StepResult, suiteName string, outputDir string) error {
	passed, failed, skipped := countResults(results)
	totalDuration := sumDuration(results)

	suite := junitTestSuite{
		Name:     suiteName,
		Tests:    passed + failed + skipped,
		Failures: failed,
		Skipped:  skipped,
		Time:     fmt.Sprintf("%.3f", totalDuration.Seconds()),
	}

	for _, r := range results {
		tc := junitTestCase{
			Name:      r.Name,
			ClassName: r.RequestName,
			Time:      fmt.Sprintf("%.3f", r.Duration.Seconds()),
		}

		if r.Skipped {
			tc.Skipped = &junitSkipped{Message: "condition not met"}
		} else if !r.Passed {
			msg := r.Error
			var content string
			for _, a := range r.Assertions {
				if !a.Passed {
					if msg == "" {
						msg = a.Message
					}
					content += fmt.Sprintf("%s\nExpected: %s\nActual: %s\n\n",
						a.Message, a.Expected, a.Actual)
				}
			}
			tc.Failure = &junitFailure{
				Message: msg,
				Type:    "AssertionError",
				Content: content,
			}
		}

		suite.Cases = append(suite.Cases, tc)
	}

	suites := junitTestSuites{Suites: []junitTestSuite{suite}}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	filename := filepath.Join(outputDir, "report-"+time.Now().Format("2006-01-02T15-04-05")+".xml")
	data, err := xml.MarshalIndent(suites, "", "  ")
	if err != nil {
		return err
	}

	output := []byte(xml.Header)
	output = append(output, data...)
	return os.WriteFile(filename, output, 0644)
}
