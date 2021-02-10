package qac

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/enr/go-files/files"
	"gopkg.in/yaml.v2"
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
	dat, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	plan := TestPlan{}
	err = yaml.Unmarshal(dat, &plan)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	basedir, _ := filepath.Abs(filepath.Dir(path))
	context := planContext{}
	context.basedir = basedir
	return l.execute(plan, context)
}

// Execute run tests loaded from a TestPlan.
func (l *Launcher) Execute(plan TestPlan) *TestExecutionReport {
	path, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	context := planContext{}
	context.basedir = path
	return l.execute(plan, context)
}

func (l *Launcher) execute(plan TestPlan, context planContext) *TestExecutionReport {
	report := &TestExecutionReport{}
	// report.phase = "plan preconditions"
	preconditions := plan.Preconditions
	proceed := l.verifyPreconditions(preconditions, context, report)
	if !proceed {
		report.addEntryInfo(`preconditions`, fmt.Sprintf("error in plan preconditions, stop plan execution"))
		return report
	}
	specs := plan.Specs
	for key, spec := range specs {
		spec.id = key
		context.currentSpec = spec
		// report.phase = fmt.Sprintf(`%s / %s`, k, spec.Command.String())
		l.executeSpec(context, report)
	}
	return report
}

func (l *Launcher) verifyPreconditions(preconditions Preconditions, context planContext, report *TestExecutionReport) bool {
	fa := preconditions.FileSystemAssertions
	var assertionResult AssertionResult
	phase := `preconditions`
	if context.currentSpec.ID() != "" {
		phase = fmt.Sprintf(`%s preconditions`, context.currentSpec.ID())
	}
	var a assertion
	var err error
	proceed := true
	for _, f := range fa {
		a, err = f.actualAssertion(context)
		if err != nil {
			report.addEntryAsError(phase, err)
			proceed = false
		}
		assertionResult = a.verify(context)
		report.addEntryAsAssertionResult(phase, assertionResult)
		if !assertionResult.Success() {
			proceed = false
		}
	}
	return proceed
}

func (l *Launcher) executeSpec(context planContext, report *TestExecutionReport) {
	spec := context.currentSpec
	preconditions := spec.Preconditions
	proceed := l.verifyPreconditions(preconditions, context, report)
	if !proceed {
		report.addEntryInfo(`preconditions`, fmt.Sprintf("error in spec %s preconditions, stop spec execution", spec.ID()))
		return
	}
	phase := fmt.Sprintf(`%s`, spec.ID())
	if spec.Description != "" {
		phase = fmt.Sprintf(`%s : %s`, phase, spec.Description)
	}
	command := spec.Command
	wd, err := resolvePath(command.WorkingDir, context)
	if err != nil {
		report.addEntryAsError(phase, err)
	}
	if !files.IsDir(wd) {
		report.addEntryAsErrorString(phase, fmt.Sprintf(`invalid working dir %s (not found or not dir)`, wd))
		fmt.Println("invalid working dir... stop spec...")
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

	var a assertion
	for _, f := range fa {
		a, err = f.actualAssertion(context)
		if err != nil {
			report.addEntryAsError(phase, err)
		}
		report.addEntryAsAssertionResult(phase, a.verify(context))
	}
}
