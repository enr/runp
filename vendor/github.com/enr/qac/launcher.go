package qac

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/enr/go-files/files"
	"gopkg.in/yaml.v3"
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

// ExecuteFile run tests loaded from a file.
func (l *Launcher) ExecuteFile(path string) *TestExecutionReport {
	report := &TestExecutionReport{}
	dat, err := os.ReadFile(path)
	if err != nil {
		report.addEntryAsError("load", asConfigError(fmt.Errorf("reading test plan %q: %w", path, err)))
		return report
	}
	plan := TestPlan{}
	dec := yaml.NewDecoder(bytes.NewReader(dat))
	dec.KnownFields(true)
	if err = dec.Decode(&plan); err != nil {
		report.addEntryAsError("load", asConfigError(fmt.Errorf("parsing test plan %q: %w", path, err)))
		return report
	}
	basedir, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		report.addEntryAsError("load", asInfraError(fmt.Errorf("resolving base directory for %q: %w", path, err)))
		return report
	}
	context := planContext{}
	context.basedir = basedir
	return l.execute(plan, context)
}

// Execute run tests loaded from a TestPlan.
func (l *Launcher) Execute(plan TestPlan) *TestExecutionReport {
	report := &TestExecutionReport{}
	basedir, err := os.Getwd()
	if err != nil {
		report.addEntryAsError("load", asInfraError(fmt.Errorf("getting working directory: %w", err)))
		return report
	}
	context := planContext{}
	context.basedir = basedir
	return l.execute(plan, context)
}

func (l *Launcher) execute(plan TestPlan, context planContext) *TestExecutionReport {
	report := &TestExecutionReport{}
	// report.phase = "plan preconditions"
	preconditions := plan.Preconditions
	proceed, failed, total := l.verifyPreconditions(preconditions, context, report)
	if !proceed {
		report.addEntryInfo("preconditions", fmt.Sprintf("%d of %d preconditions failed, stopping plan execution", failed, total))
		return report
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
		context.currentSpec = spec
		phase := specPhase(spec)
		start := time.Now()
		report.openBlock(phase, i+1, numSpecs, start)
		l.executeSpec(context, report)
		report.closeBlock(phase, time.Since(start))
	}
	return report
}

func specPhase(spec Spec) string {
	if spec.Description != "" {
		return spec.ID() + " : " + spec.Description
	}
	return spec.ID()
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
			report.addEntryAsError(phase, err)
			failed++
			continue
		}
		result := a.verify(context)
		report.addEntryAsAssertionResult(phase, result)
		if !result.Success() {
			failed++
		}
	}
	return failed == 0, failed, total
}

func (l *Launcher) executeSpec(context planContext, report *TestExecutionReport) {
	spec := context.currentSpec
	phase := specPhase(spec)
	preconditions := spec.Preconditions
	proceed, failed, total := l.verifyPreconditions(preconditions, context, report)
	if !proceed {
		report.addEntrySkipped(phase, fmt.Sprintf("skipped: %d of %d preconditions failed", failed, total))
		return
	}
	command := spec.Command
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
	context.commandResult = l.executor.execute(command)
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
}
