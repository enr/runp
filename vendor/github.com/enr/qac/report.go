package qac

import (
	"fmt"
	"testing"
)

// ReportEntryType represents the type of a report entry.
type ReportEntryType int8

const (
	reportTypeNone ReportEntryType = iota
	// ErrorType is report entry error
	ErrorType
	// InfoType is report entry info
	InfoType
	// SuccessType is report entry success
	SuccessType
)

// ReportEntry is a single unit of information in a report.
type ReportEntry struct {
	kind        ReportEntryType
	description string
	errors      []error
}

func (r *ReportEntry) addError(err error) {
	r.errors = append(r.errors, err)
}

// Description is the textual representation of a report entry.
func (r *ReportEntry) Description() string {
	return r.description
}

// Errors returns the errors list in a report entry.
func (r *ReportEntry) Errors() []error {
	return r.errors
}

// Kind returns the type of a report entry: error, info, success...
func (r *ReportEntry) Kind() ReportEntryType {
	return r.kind
}

// ReportBlock is an aggregate of report entries classified on the phase.
type ReportBlock struct {
	phase   string
	entries []ReportEntry
}

// Phase returns the phase of a block.
func (r *ReportBlock) Phase() string {
	return r.phase
}

// Entries returns the entries of a block.
func (r *ReportBlock) Entries() []ReportEntry {
	return r.entries
}

func newReportEntryFromAssertionResult(ar AssertionResult) ReportEntry {
	k := ErrorType
	if ar.Success() {
		k = SuccessType
	}
	return ReportEntry{description: ar.description, kind: k, errors: ar.errors}
}
func newReportEntryFromError(err error) ReportEntry {
	return ReportEntry{description: `error`, kind: ErrorType, errors: []error{err}}
}
func newReportEntryInfo(msg string) ReportEntry {
	return ReportEntry{description: msg, kind: InfoType, errors: []error{}}
}

// TestExecutionReport is the full report on a test execution
type TestExecutionReport struct {
	// used to keep report entries ordered
	blocks []*ReportBlock
}

func (r *TestExecutionReport) addEntryAsErrorString(phase string, message string) {
	r.addEntryAsError(phase, fmt.Errorf(message))
}

func (r *TestExecutionReport) addEntryAsError(phase string, err error) {
	entry := newReportEntryFromError(err)
	r.addEntry(phase, entry)
}

func (r *TestExecutionReport) addEntryAsAssertionResult(phase string, ar AssertionResult) {
	entry := newReportEntryFromAssertionResult(ar)
	r.addEntry(phase, entry)
}
func (r *TestExecutionReport) addEntryInfo(phase string, msg string) {
	entry := newReportEntryInfo(msg)
	r.addEntry(phase, entry)
}

func (r *TestExecutionReport) addEntry(phase string, entry ReportEntry) {
	for _, block := range r.blocks {
		if block.phase == phase {
			block.entries = append(block.entries, entry)
			return
		}
	}
	r.blocks = append(r.blocks, &ReportBlock{phase: phase, entries: []ReportEntry{entry}})
}

// Blocks returns the blocks list in a full report.
func (r *TestExecutionReport) Blocks() []*ReportBlock {
	return r.blocks
}

// AllErrors returns all errors in a report, without considering blocks or phases.
func (r *TestExecutionReport) AllErrors() []error {
	errors := []error{}
	for _, block := range r.Blocks() {
		for _, entry := range block.Entries() {
			for _, err := range entry.Errors() {
				errors = append(errors, err)
			}
		}
	}
	return errors
}

// Reporter is the interface for components publishing the report.
type Reporter interface {
	Publish(report *TestExecutionReport) error
}

// NewTestLogsReporter returns a Reporter implementation using the testing log.
func NewTestLogsReporter(t *testing.T) Reporter {
	return &testLogsReporter{t: t}
}

type testLogsReporter struct {
	t *testing.T
}

func (r *testLogsReporter) Publish(report *TestExecutionReport) error {

	for _, block := range report.Blocks() {
		r.t.Logf("Phase <%s>", block.Phase())
		for _, entry := range block.Entries() {
			// r.ui.Lifecyclef(" - %s", entry.Description())
			switch entry.Kind() {
			case ErrorType:
				r.t.Logf("  | - KO %s", entry.Description())
				for i, err := range entry.Errors() {
					r.t.Logf("    %d %s", (i + 1), err.Error())
				}
				break
			case InfoType:
				r.t.Logf("  | INFO %s", entry.Description())
				break
			case SuccessType:
				r.t.Logf("  | - OK %s", entry.Description())
				break
			default:
				r.t.Logf("unexpected kind %v", entry.Kind())
			}
		}
	}
	return nil
}

// NewConsoleReporter returns a Reporter implementation writing to the stdout.
func NewConsoleReporter() Reporter {
	return &consoleReporter{}
}

type consoleReporter struct {
}

func (r *consoleReporter) Publish(report *TestExecutionReport) error {
	for _, block := range report.Blocks() {
		fmt.Printf("Phase <%s>\n", block.Phase())
		for _, entry := range block.Entries() {
			// r.ui.Lifecyclef(" - %s", entry.Description())
			switch entry.Kind() {
			case ErrorType:
				fmt.Printf("  | - KO %s\n", entry.Description())
				for i, err := range entry.Errors() {
					fmt.Printf("      (%d) %s\n", (i + 1), err.Error())
				}
				break
			case InfoType:
				fmt.Printf("  | INFO %s\n", entry.Description())
				break
			case SuccessType:
				fmt.Printf("  | - OK %s\n", entry.Description())
				break
			default:
				fmt.Printf("unexpected kind %v\n", entry.Kind())
			}
		}
	}
	return nil
}
