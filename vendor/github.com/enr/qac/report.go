package qac

import (
	"fmt"
	"testing"
	"time"
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
	// SkippedType means the spec was not executed because preconditions were not met
	SkippedType
	// TimedOutType means the command exceeded its configured timeout
	TimedOutType
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

// String returns a human-readable name for the entry type.
func (t ReportEntryType) String() string {
	switch t {
	case ErrorType:
		return "ERROR"
	case InfoType:
		return "INFO"
	case SuccessType:
		return "OK"
	case SkippedType:
		return "SKIP"
	case TimedOutType:
		return "TIMEOUT"
	default:
		return "UNKNOWN"
	}
}

// Kind returns the type of a report entry: error, info, success, skipped...
func (r *ReportEntry) Kind() ReportEntryType {
	return r.kind
}

// ReportBlock is an aggregate of report entries classified on the phase.
type ReportBlock struct {
	phase     string
	index     int // 1-based ordinal among specs; 0 for non-spec blocks
	total     int
	startedAt time.Time
	duration  time.Duration
	entries   []ReportEntry
}

// Phase returns the phase of a block.
func (r *ReportBlock) Phase() string { return r.phase }

// Entries returns the entries of a block.
func (r *ReportBlock) Entries() []ReportEntry { return r.entries }

// Index returns the 1-based ordinal of this spec in the execution order (0 for non-spec blocks).
func (r *ReportBlock) Index() int { return r.index }

// Total returns the total number of specs in the plan (0 for non-spec blocks).
func (r *ReportBlock) Total() int { return r.total }

// StartedAt returns when execution of this block started.
func (r *ReportBlock) StartedAt() time.Time { return r.startedAt }

// Duration returns how long this block took to execute.
func (r *ReportBlock) Duration() time.Duration { return r.duration }

// TimedOut returns true if the command in this block exceeded its configured timeout.
func (r *ReportBlock) TimedOut() bool {
	for _, e := range r.entries {
		if e.Kind() == TimedOutType {
			return true
		}
	}
	return false
}

// Failed returns true if this block contains at least one error or timeout entry.
func (r *ReportBlock) Failed() bool {
	for _, e := range r.entries {
		if e.Kind() == ErrorType || e.Kind() == TimedOutType {
			return true
		}
	}
	return false
}

// Skipped returns true if this spec block was skipped (by skip field, skip_if, or tag filter).
func (r *ReportBlock) Skipped() bool {
	if len(r.entries) == 0 && r.index > 0 {
		return true
	}
	for _, e := range r.entries {
		if e.Kind() == SkippedType {
			return true
		}
	}
	return false
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

func newReportEntrySkipped(reason string) ReportEntry {
	return ReportEntry{description: reason, kind: SkippedType, errors: []error{}}
}

func newReportEntryTimedOut(timeout string) ReportEntry {
	msg := fmt.Sprintf("command timed out after %s", timeout)
	return ReportEntry{description: msg, kind: TimedOutType, errors: []error{fmt.Errorf("%s", msg)}}
}

// TestExecutionReport is the full report on a test execution
type TestExecutionReport struct {
	blocks     []*ReportBlock
	blockIndex map[string]*ReportBlock
}

func (r *TestExecutionReport) addEntryAsErrorString(phase string, message string) {
	r.addEntryAsError(phase, fmt.Errorf("%s", message))
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

func (r *TestExecutionReport) addEntrySkipped(phase string, reason string) {
	entry := newReportEntrySkipped(reason)
	r.addEntry(phase, entry)
}

func (r *TestExecutionReport) addEntryTimedOut(phase string, timeout string) {
	entry := newReportEntryTimedOut(timeout)
	r.addEntry(phase, entry)
}

func (r *TestExecutionReport) addEntry(phase string, entry ReportEntry) {
	if b, ok := r.blockIndex[phase]; ok {
		b.entries = append(b.entries, entry)
		return
	}
	b := &ReportBlock{phase: phase, entries: []ReportEntry{entry}}
	r.blocks = append(r.blocks, b)
	r.ensureIndex()[phase] = b
}

func (r *TestExecutionReport) ensureIndex() map[string]*ReportBlock {
	if r.blockIndex == nil {
		r.blockIndex = make(map[string]*ReportBlock)
	}
	return r.blockIndex
}

// openBlock pre-creates a numbered spec block before execution begins,
// so index/total/startedAt are available even if the block ends up with no entries.
func (r *TestExecutionReport) openBlock(phase string, index, total int, startedAt time.Time) {
	if b, ok := r.blockIndex[phase]; ok {
		b.index = index
		b.total = total
		b.startedAt = startedAt
		return
	}
	b := &ReportBlock{phase: phase, index: index, total: total, startedAt: startedAt}
	r.blocks = append(r.blocks, b)
	r.ensureIndex()[phase] = b
}

// closeBlock stamps the duration onto the block once execution is done.
func (r *TestExecutionReport) closeBlock(phase string, d time.Duration) {
	if b, ok := r.blockIndex[phase]; ok {
		b.duration = d
	}
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

// Success returns true when no block recorded an error or timeout.
func (r *TestExecutionReport) Success() bool {
	return len(r.AllErrors()) == 0
}

// FailedSpecs returns the phase names of spec blocks that failed.
func (r *TestExecutionReport) FailedSpecs() []string {
	var names []string
	for _, b := range r.blocks {
		if b.Index() > 0 && b.Failed() {
			names = append(names, b.Phase())
		}
	}
	return names
}

// Summary returns a one-line human-readable description of the execution
// result, e.g. "3/5 specs passed" or "5/5 specs passed (2 skipped)".
func (r *TestExecutionReport) Summary() string {
	total, passed, skipped := 0, 0, 0
	for _, b := range r.blocks {
		if b.Index() == 0 {
			continue
		}
		total++
		if b.Skipped() {
			skipped++
		} else if !b.Failed() {
			passed++
		}
	}
	s := fmt.Sprintf("%d/%d specs passed", passed, total)
	if skipped > 0 {
		s += fmt.Sprintf(" (%d skipped)", skipped)
	}
	return s
}

// FailWith calls t.Errorf for every error in the report, prefixed with the
// block phase so failures are easy to locate. It is the idiomatic one-liner
// to fail a Go test when a qac plan has errors:
//
//	report.FailWith(t)
func (r *TestExecutionReport) FailWith(t *testing.T) {
	t.Helper()
	for _, block := range r.blocks {
		for _, entry := range block.Entries() {
			for _, err := range entry.Errors() {
				t.Errorf("[%s] %v", block.Phase(), err)
			}
		}
	}
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
		label := block.Phase()
		if block.Index() > 0 {
			label = fmt.Sprintf("[%d/%d] %s", block.Index(), block.Total(), block.Phase())
		}
		status := blockStatus(block)
		if block.Duration() > 0 {
			r.t.Logf("%-40s (%s) %s", label, block.Duration().Round(time.Millisecond), status)
		} else {
			r.t.Logf("%-40s %s", label, status)
		}
		for _, entry := range block.Entries() {
			switch entry.Kind() {
			case ErrorType:
				r.t.Logf("  | KO %s", entry.Description())
				for i, err := range entry.Errors() {
					r.t.Logf("      %d. %s", i+1, err.Error())
				}
			case SkippedType:
				r.t.Logf("  | SKIP %s", entry.Description())
			case InfoType:
				r.t.Logf("  | INFO %s", entry.Description())
			case TimedOutType:
				r.t.Logf("  | TIMEOUT %s", entry.Description())
			}
		}
	}
	return nil
}

// NewConsoleReporter returns a Reporter implementation writing to the stdout.
func NewConsoleReporter() Reporter {
	return &consoleReporter{}
}

type consoleReporter struct{}

func (r *consoleReporter) Publish(report *TestExecutionReport) error {
	for _, block := range report.Blocks() {
		label := block.Phase()
		if block.Index() > 0 {
			label = fmt.Sprintf("[%d/%d] %s", block.Index(), block.Total(), block.Phase())
		}
		status := blockStatus(block)
		if block.Duration() > 0 {
			fmt.Printf("%-40s (%s) %s\n", label, block.Duration().Round(time.Millisecond), status)
		} else {
			fmt.Printf("%-40s %s\n", label, status)
		}
		for _, entry := range block.Entries() {
			switch entry.Kind() {
			case ErrorType:
				fmt.Printf("  | KO %s\n", entry.Description())
				for i, err := range entry.Errors() {
					fmt.Printf("      %d. %s\n", i+1, err.Error())
				}
			case SkippedType:
				fmt.Printf("  | SKIP %s\n", entry.Description())
			case InfoType:
				fmt.Printf("  | INFO %s\n", entry.Description())
			case TimedOutType:
				fmt.Printf("  | TIMEOUT %s\n", entry.Description())
			}
		}
	}
	return nil
}

func blockStatus(block *ReportBlock) string {
	for _, entry := range block.Entries() {
		if entry.Kind() == ErrorType || entry.Kind() == TimedOutType {
			return "KO"
		}
	}
	for _, entry := range block.Entries() {
		if entry.Kind() == SkippedType {
			return "SKIP"
		}
	}
	if len(block.Entries()) == 0 && block.Index() > 0 {
		return "SKIP"
	}
	return "OK"
}
