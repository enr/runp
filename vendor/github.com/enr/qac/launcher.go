package qac

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/enr/go-files/files"
)

type planContext struct {
	basedir       string
	commandResult executionResult
	currentSpec   Spec
}

// useful for tests
func newLauncher(e executor) *Launcher {
	return &Launcher{
		executor: e,
	}
}

// NewLauncher creates a default implementation for Launcher.
func NewLauncher() *Launcher {
	return &Launcher{
		executor: &runcmdExecutor{},
	}
}

// Launcher checks the results respect expectations.
type Launcher struct {
	executor executor
}

// ExecuteFile loads a test plan from path and runs it.
// Returns a non-nil error when the file cannot be read or parsed; spec-level
// failures are recorded in the report and do not cause an error return.
func (l *Launcher) ExecuteFile(path string, opts ...RunOption) (*TestExecutionReport, error) {
	cfg := applyOptions(opts)
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolving path %q: %w", path, err)
	}
	plan, err := loadPlanFile(absPath, make(map[string]bool))
	if err != nil {
		return nil, fmt.Errorf("parsing test plan %q: %w", path, err)
	}
	context := planContext{basedir: filepath.Dir(absPath)}
	return l.execute(plan, context, cfg), nil
}

// ExecuteFileT loads the YAML plan at path, runs all specs, and integrates
// with Go's testing.TB: load and parse errors call t.Fatalf (stopping the
// test immediately); each spec failure calls t.Errorf. The returned report
// is never nil.
// Pass RunOption values (WithTags, SkipTags, FailFast) to filter specs.
// It is the idiomatic single-line form for Go tests:
//
//	qac.NewLauncher().ExecuteFileT(t, "plan.yaml")
func (l *Launcher) ExecuteFileT(t testing.TB, path string, opts ...RunOption) *TestExecutionReport {
	t.Helper()
	report, err := l.ExecuteFile(path, opts...)
	if err != nil {
		t.Fatalf("qac: cannot load plan: %v", err)
		return nil
	}
	for _, e := range report.AllErrors() {
		t.Errorf("qac: %v", e)
	}
	return report
}

// ExecuteT runs plan and integrates with Go's testing.TB: each spec failure
// calls t.Errorf. The returned report is never nil.
// Pass RunOption values (WithTags, SkipTags, FailFast) to filter specs.
// It is the idiomatic single-line form when constructing plans in Go:
//
//	qac.NewLauncher().ExecuteT(t, plan)
func (l *Launcher) ExecuteT(t testing.TB, plan TestPlan, opts ...RunOption) *TestExecutionReport {
	t.Helper()
	report := l.Execute(plan, opts...)
	for _, e := range report.AllErrors() {
		t.Errorf("qac: %v", e)
	}
	return report
}

// Execute runs all specs in plan and returns a report of the results.
// The report is never nil; spec-level failures are recorded inside it and do
// not cause an error return. Pass RunOption values (WithTags, SkipTags,
// FailFast) to control which specs run.
// For use in Go tests prefer ExecuteT, which forwards spec failures to t.Errorf.
func (l *Launcher) Execute(plan TestPlan, opts ...RunOption) *TestExecutionReport {
	cfg := applyOptions(opts)
	report := &TestExecutionReport{}
	basedir, err := os.Getwd()
	if err != nil {
		report.addEntryAsError("load", asInfraError(fmt.Errorf("getting working directory: %w", err)))
		return report
	}
	context := planContext{basedir: basedir}
	return l.execute(plan, context, cfg)
}

func (l *Launcher) execute(plan TestPlan, context planContext, cfg runConfig) *TestExecutionReport {
	report := &TestExecutionReport{}

	// Plan teardown always runs at the end, even if preconditions, setup or specs fail.
	if len(plan.Teardown) > 0 {
		defer l.runCommands(plan.Teardown, "teardown", context, report, false)
	}

	preconditions := plan.Preconditions
	proceed, _, _ := l.verifyPreconditions(preconditions, context, report)
	if !proceed {
		report.addEntryInfo("preconditions", "plan execution stopped")
		return report
	}

	// Plan setup: stop specs (but not teardown) on first failure.
	if len(plan.Setup) > 0 {
		if !l.runCommands(plan.Setup, "setup", context, report, true) {
			return report
		}
	}

	order := plan.specOrder
	if len(order) == 0 {
		for key := range plan.Specs {
			order = append(order, key)
		}
	}
	numSpecs := len(order)
	for i, key := range order {
		spec := plan.Specs[key]
		spec.id = key
		phase := specPhase(spec)
		start := time.Now()
		report.openBlock(phase, i+1, numSpecs, start)
		skipped := false
		if reason := tagSkipReason(spec, cfg); reason != "" {
			report.addEntrySkipped(phase, reason)
			skipped = true
		} else {
			context.currentSpec = spec
			l.executeSpecWithRetries(context, report)
		}
		report.closeBlock(phase, time.Since(start))
		if cfg.failFast && !skipped {
			if b, ok := report.blockIndex[phase]; ok && b.Failed() {
				return report
			}
		}
	}
	return report
}

// runCommands runs each command in order, reporting failures into the given phase.
// If stopOnFailure is true it returns false on the first failed command; otherwise
// it runs all commands and returns whether all succeeded.
func (l *Launcher) runCommands(commands []Command, phase string, ctx planContext, report *TestExecutionReport, stopOnFailure bool) bool {
	allOk := true
	for _, cmd := range commands {
		if cmd.Cli != "" && cmd.Exe != "" {
			report.addEntryAsError(phase, asConfigError(fmt.Errorf("cli and exe are mutually exclusive")))
			allOk = false
			if stopOnFailure {
				return false
			}
			continue
		}
		if cmd.Stdin != "" && cmd.StdinFile != "" {
			report.addEntryAsError(phase, asConfigError(fmt.Errorf("stdin and stdin_file are mutually exclusive")))
			allOk = false
			if stopOnFailure {
				return false
			}
			continue
		}
		if cmd.StdinFile != "" {
			resolved, err := resolvePath(cmd.StdinFile, ctx)
			if err != nil {
				report.addEntryAsError(phase, asConfigError(fmt.Errorf("resolving stdin_file %q: %w", cmd.StdinFile, err)))
				allOk = false
				if stopOnFailure {
					return false
				}
				continue
			}
			cmd.StdinFile = resolved
		}
		wd, err := resolvePath(cmd.WorkingDir, ctx)
		if err != nil {
			report.addEntryAsError(phase, asInfraError(err))
			allOk = false
			if stopOnFailure {
				return false
			}
			continue
		}
		if !files.IsDir(wd) {
			report.addEntryAsError(phase, asInfraError(fmt.Errorf("invalid working dir %s (not found or not dir)", wd)))
			allOk = false
			if stopOnFailure {
				return false
			}
			continue
		}
		cmd.WorkingDir = wd
		result := l.executor.execute(cmd)
		if result.timedOut {
			report.addEntryTimedOut(phase, cmd.Timeout)
			allOk = false
			if stopOnFailure {
				return false
			}
		} else if !result.success {
			msg := fmt.Sprintf("command failed (exit %d): %s", result.exitCode, result.execution)
			if s := strings.TrimSpace(result.stderr); s != "" {
				msg += "\nstderr: " + s
			}
			report.addEntryAsError(phase, fmt.Errorf("%s", msg))
			allOk = false
			if stopOnFailure {
				return false
			}
		}
	}
	return allOk
}

func specPhase(spec Spec) string {
	if spec.Description != "" {
		return spec.ID() + " : " + spec.Description
	}
	return spec.ID()
}

// executeSpecWithRetries runs executeSpec up to spec.Retries+1 times.
// Each attempt is isolated in a temporary report; only the decisive attempt's
// entries (success, or last failure) are merged into the main report.
// Between attempts an info entry is added and retry_delay is observed.
// Spec teardown runs on every attempt (clean state for next retry) but only
// the final attempt's teardown entries appear in the main report.
func (l *Launcher) executeSpecWithRetries(ctx planContext, report *TestExecutionReport) {
	spec := ctx.currentSpec
	phase := specPhase(spec)
	maxAttempts := 1
	if spec.Retries > 0 {
		maxAttempts = spec.Retries + 1
	}

	var retryDelay time.Duration
	if spec.RetryDelay != "" {
		d, err := time.ParseDuration(spec.RetryDelay)
		if err != nil {
			report.addEntryAsError(phase, asConfigError(fmt.Errorf("invalid retry_delay %q: %w", spec.RetryDelay, err)))
		} else {
			retryDelay = d
		}
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		tmp := &TestExecutionReport{}
		l.executeSpec(ctx, tmp)

		var failed bool
		if b, ok := tmp.blockIndex[phase]; ok {
			failed = b.Failed()
		}
		isLast := attempt == maxAttempts

		if !failed || isLast {
			for _, b := range tmp.Blocks() {
				for _, e := range b.entries {
					report.addEntry(b.phase, e)
				}
			}
			if attempt > 1 && !failed {
				report.addEntryInfo(phase, fmt.Sprintf("passed on attempt %d of %d", attempt, maxAttempts))
			}
			return
		}

		report.addEntryInfo(phase, fmt.Sprintf("attempt %d of %d failed, retrying...", attempt, maxAttempts))
		if retryDelay > 0 {
			time.Sleep(retryDelay)
		}
	}
}

func (l *Launcher) verifyPreconditions(preconditions Preconditions, context planContext, report *TestExecutionReport) (bool, int, int) {
	fa := preconditions.FileSystemAssertions
	phase := `preconditions`
	if context.currentSpec.ID() != "" {
		phase = fmt.Sprintf(`%s preconditions`, context.currentSpec.ID())
	}
	failed := 0
	total := len(fa)
	for _, f := range fa {
		a, err := f.actualAssertion(context)
		if err != nil {
			report.addEntryAsError(phase, fmt.Errorf("precondition failed: %w", err))
			failed++
			continue
		}
		result := a.verify(context)
		if result.Success() {
			report.addEntryAsAssertionResult(phase, result)
		} else {
			for _, e := range result.Errors() {
				report.addEntryAsError(phase, fmt.Errorf("precondition failed: %s", e.Error()))
			}
			failed++
		}
	}
	return failed == 0, failed, total
}

func (l *Launcher) executeSpec(context planContext, report *TestExecutionReport) {
	spec := context.currentSpec
	phase := specPhase(spec)
	if reason := specSkipReason(spec); reason != "" {
		report.addEntrySkipped(phase, reason)
		return
	}
	preconditions := spec.Preconditions
	proceed, _, _ := l.verifyPreconditions(preconditions, context, report)
	if !proceed {
		report.addEntrySkipped(phase, "skipped: preconditions not met")
		return
	}
	// Spec teardown always runs from this point on, even if setup or the command fail.
	if len(spec.Teardown) > 0 {
		defer l.runCommands(spec.Teardown, phase+" teardown", context, report, false)
	}
	// Spec setup: run before the command; stop (but not teardown) on first failure.
	if len(spec.Setup) > 0 {
		if !l.runCommands(spec.Setup, phase+" setup", context, report, true) {
			return
		}
	}
	command := spec.Command
	if command.Cli != "" && command.Exe != "" {
		report.addEntryAsError(phase, asConfigError(fmt.Errorf("cli and exe are mutually exclusive")))
		return
	}
	if command.Stdin != "" && command.StdinFile != "" {
		report.addEntryAsError(phase, asConfigError(fmt.Errorf("stdin and stdin_file are mutually exclusive")))
		return
	}
	if command.StdinFile != "" {
		resolved, err := resolvePath(command.StdinFile, context)
		if err != nil {
			report.addEntryAsError(phase, asConfigError(fmt.Errorf("resolving stdin_file %q: %w", command.StdinFile, err)))
			return
		}
		command.StdinFile = resolved
	}
	wd, err := resolvePath(command.WorkingDir, context)
	if err != nil {
		report.addEntryAsError(phase, asInfraError(err))
		return
	}
	if !files.IsDir(wd) {
		report.addEntryAsError(phase, asInfraError(fmt.Errorf("invalid working dir %s (not found or not dir)", wd)))
		return
	}
	command.WorkingDir = wd
	cmdStart := time.Now()
	context.commandResult = l.executor.execute(command)
	context.commandResult.duration = time.Since(cmdStart)
	if context.commandResult.timedOut {
		report.addEntryTimedOut(phase, command.Timeout)
		return
	}
	expectations := spec.Expectations
	report.addEntryAsAssertionResult(phase, expectations.StatusAssertion.verify(context))
	oa := expectations.OutputAssertions.Stdout
	oa.id = `stdout`
	report.addEntryAsAssertionResult(phase, oa.verify(context))
	ea := expectations.OutputAssertions.Stderr
	ea.id = `stderr`
	report.addEntryAsAssertionResult(phase, ea.verify(context))
	fa := expectations.FileSystemAssertions

	for _, f := range fa {
		a, err := f.actualAssertion(context)
		if err != nil {
			report.addEntryAsError(phase, err)
			continue
		}
		report.addEntryAsAssertionResult(phase, a.verify(context))
	}
	da := expectations.DurationAssertion
	if da.Max != "" || da.Min != "" {
		report.addEntryAsAssertionResult(phase, da.verify(context))
	}
}
