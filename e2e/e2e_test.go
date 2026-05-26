package e2e

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/enr/go-files/files"
	"github.com/enr/qac"
)

func TestExecutions(t *testing.T) {
	executions(`./basic.yaml`, t)
	executions(`./completion.yaml`, t)
	executions(`./process.yaml`, t)
	executions(`./validate.yaml`, t)
	os := runtime.GOOS
	osTestFile := fmt.Sprintf(`./%s.yaml`, os)
	if files.Exists(osTestFile) {
		executions(osTestFile, t)
	}
}

func executions(f string, t *testing.T) {
	launcher := qac.NewLauncher()
	report, err := launcher.ExecuteFile(f)
	if err != nil {
		t.Fatalf("ExecuteFile %s: %v", f, err)
	}
	reporter := qac.NewTestLogsReporter(t)
	reporter.Publish(report)

	for ei, err := range report.AllErrors() {
		t.Errorf(`%d %s error %v`, ei, f, err)
	}
}
