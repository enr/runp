package core

import (
	"context"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/bww/impatient"
)

// NewExecutor creates new RunpfileExecutor
func NewExecutor(rf *Runpfile) *RunpfileExecutor {
	pidDir := ""
	if rf.Root != "" {
		if d, err := PIDDirForRoot(rf.Root); err == nil {
			pidDir = d
		}
	}
	return &RunpfileExecutor{
		rf:                  rf,
		LoggerFactory:       createProcessLogger,
		environmentSettings: loadEnvironmentSettings(),
		newPipe:             os.Pipe,
		PIDDir:              pidDir,
	}
}

// RunpfileExecutor Executor implementation for Runpfile.
type RunpfileExecutor struct {
	rf                  *Runpfile
	LoggerFactory       func(string, int, LoggerConfig) Logger
	longest             int
	environmentSettings *EnvironmentSettings
	newPipe             func() (*os.File, *os.File, error)
	PIDDir              string
}

func (e *RunpfileExecutor) longestName() int {
	if e.longest > 0 {
		return e.longest
	}
	ln := 0
	for _, process := range e.rf.Units {
		if len(process.Name) > ln {
			ln = len(process.Name)
		}
	}
	e.longest = ln
	return e.longest
}

func (e *RunpfileExecutor) initializeUnits() {
	for _, unit := range e.rf.Units {
		unit.vars = e.rf.Vars
		unit.secretKey = e.rf.SecretKey
		unit.environmentSettings = e.environmentSettings
		unit.process = nil
		if unit.Host != nil {
			unit.Host.vars = unit.vars
			unit.Host.secretKey = unit.secretKey
			unit.Host.stopTimeout = unit.StopTimeout
			unit.Host.pidDir = e.PIDDir
			unit.Host.environmentSettings = e.environmentSettings
		}
		if unit.Container != nil {
			unit.Container.vars = unit.vars
			unit.Container.secretKey = unit.secretKey
			unit.Container.stopTimeout = unit.StopTimeout
			unit.Container.environmentSettings = e.environmentSettings
		}
		if unit.SSHTunnel != nil {
			unit.SSHTunnel.vars = unit.vars
			unit.SSHTunnel.secretKey = unit.secretKey
			unit.SSHTunnel.stopTimeout = unit.StopTimeout
			unit.SSHTunnel.environmentSettings = e.environmentSettings
		}
		// Propagate unit-level preconditions to the process. LoadRunpfileFromPath
		// does this during loading, but units created directly (e.g. in tests or
		// via StartSingleUnit) rely on this path to have preconditions applied.
		if p := unit.Process(); p != nil {
			p.SetPreconditions(unit.Preconditions)
		}
	}
}

func (e *RunpfileExecutor) skippedUnits() map[string]bool {
	skipped := make(map[string]bool)
	for _, unit := range e.rf.Units {
		if pr := e.unitPreconditions(unit); pr != nil && pr.Vote != Proceed {
			skipped[unit.Name] = true
			ui.WriteLinef("Preconditions not satisfied for unit %s (%s): %v", unit.Name, pr.Vote, pr.Reasons)
		}
	}
	return skipped
}

func (e *RunpfileExecutor) unitPreconditions(unit *RunpUnit) *PreconditionVerifyResult {
	if unit.Host != nil {
		pr := unit.Host.VerifyPreconditions()
		return &pr
	}
	if unit.Container != nil {
		pr := unit.Container.VerifyPreconditions()
		return &pr
	}
	if unit.SSHTunnel != nil {
		pr := unit.SSHTunnel.VerifyPreconditions()
		return &pr
	}
	return nil
}

// StartSingleUnit starts only the named unit and blocks until it exits.
// Used by runp reload to restart a unit in a separate shell while
// runp up is running in another terminal.
func (e *RunpfileExecutor) StartSingleUnit(unitName string) error {
	unit, ok := e.rf.Units[unitName]
	if !ok {
		return fmt.Errorf("unit %q not found in Runpfile", unitName)
	}
	e.initializeUnits()
	if pr := e.unitPreconditions(unit); pr != nil && pr.Vote != Proceed {
		return fmt.Errorf("preconditions not satisfied for unit %q: %v", unitName, pr.Reasons)
	}
	warnInsecureSSHTunnels(e.rf.Units, map[string]bool{})
	return e.startUnit(unit)
}

// Start call start on all processes.
// Units are started in topological order derived from their depends_on fields:
// units in the same dependency layer start concurrently; each layer waits for
// the previous one to complete before beginning.
func (e *RunpfileExecutor) Start() error {
	e.initializeUnits()
	skipped := e.skippedUnits()
	if len(skipped) > 0 {
		names := make([]string, 0, len(skipped))
		for name := range skipped {
			names = append(names, name)
		}
		sort.Strings(names)
		ui.WriteLinef("Units skipped due to unsatisfied preconditions: %v", names)
		for _, name := range names {
			ui.WriteLinef("Skipping unit: %s", name)
		}
	}
	warnInsecureSSHTunnels(e.rf.Units, skipped)

	layers, err := TopologicalLayers(e.rf.Units, skipped)
	if err != nil {
		return fmt.Errorf("cannot start: %w", err)
	}

	var mu sync.Mutex
	var errs []error

	for _, layer := range layers {
		var wg sync.WaitGroup
		for _, name := range layer {
			unit := e.rf.Units[name]
			wg.Add(1)
			go func(u *RunpUnit) {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						ui.WriteLinef("Panic in goroutine for unit %s: %v", u.Name, r)
						mu.Lock()
						errs = append(errs, fmt.Errorf("panic in goroutine for unit %s: %v", u.Name, r))
						mu.Unlock()
						GetApplicationContext().TriggerShutdown()
					}
				}()
				if startErr := e.startUnit(u); startErr != nil {
					mu.Lock()
					errs = append(errs, startErr)
					mu.Unlock()
				}
			}(unit)
		}
		wg.Wait()
		if len(errs) > 0 {
			return fmt.Errorf("%d unit(s) failed to start", len(errs))
		}
	}

	return nil
}

func (e *RunpfileExecutor) startUnit(unit *RunpUnit) error {
	logger := e.LoggerFactory(unit.Name, e.longestName(), processLoggerConfiguration)
	process := unit.Process()
	logger.WriteLinef("Starting unit %s (working directory: %s)", unit.Name, process.Dir())

	appContext := GetApplicationContext()
	appContext.RegisterRunningProcess(process)

	cmd, err := e.setupProcessCommand(unit, process, logger, appContext)
	if err != nil {
		return err
	}

	if err := e.handleAwaitResources(process, logger, appContext); err != nil {
		return err
	}

	logger.Debugf("Command for process %s: %v", process.ID(), cmd)

	if err := e.verifyProcessStartability(process, logger, appContext); err != nil {
		return err
	}

	r, w, err := e.newPipe()
	if err != nil {
		return fmt.Errorf("os.Pipe: %w", err)
	}
	cmd.Stdout(w)
	cmd.Stderr(w)

	var pwg sync.WaitGroup
	pwg.Add(1)

	if err := e.startProcessCommand(cmd, unit, process, logger, appContext, w, &pwg); err != nil {
		return err
	}

	w.Close()
	e.monitorProcessExit(cmd, process, logger, appContext, &pwg)
	e.readProcessOutput(r, process, logger)
	pwg.Wait()
	return nil
}

func (e *RunpfileExecutor) setupProcessCommand(unit *RunpUnit, process RunpProcess, logger Logger, appContext *ApplicationContext) (RunpCommand, error) {
	cmd, err := process.StartCommand()
	if err != nil {
		logger.WriteLinef("Failed to build command for unit %s: %v", unit.Name, err)
		appContext.AddReport(err.Error())
		appContext.RemoveRunningProcess(process)
		return nil, err
	}
	return cmd, nil
}

func (e *RunpfileExecutor) startProcessCommand(cmd RunpCommand, unit *RunpUnit, process RunpProcess, logger Logger, appContext *ApplicationContext, w *os.File, pwg *sync.WaitGroup) error {
	if err := process.PreStart(); err != nil {
		w.Close()
		logger.WriteLinef("Pre-start hook failed for unit %s: %v", unit.Name, err)
		appContext.RemoveRunningProcess(process)
		pwg.Done()
		return err
	}
	err := cmd.Start()
	if err != nil {
		w.Close()
		logger.WriteLinef("Failed to start process %s: starting process %s: %v", unit.Name, unit.Name, err)
		appContext.RemoveRunningProcess(process)
		pwg.Done()
		return err
	}
	logger.Debugf("Process %s started successfully", process.ID())
	process.OnStarted(cmd.Pid())
	return nil
}

func (e *RunpfileExecutor) handleAwaitResources(process RunpProcess, logger Logger, appContext *ApplicationContext) error {
	if !process.ShouldWait() {
		return nil
	}

	start := time.Now()
	resources := []string{}
	if process.AwaitResource() != "" {
		resources = append(resources, process.AwaitResource())
	}

	duration, err := time.ParseDuration(process.AwaitTimeout())
	if err != nil {
		logger.WriteLinef("Invalid await timeout duration format '%s': %v", process.AwaitTimeout(), err)
		appContext.AddReport(err.Error())
		appContext.RemoveRunningProcess(process)
		return err
	}

	err = await(duration, resources)
	if err != nil {
		if err == impatient.ErrTimeout {
			logger.WriteLinef("Timeout exceeded while awaiting resources for process %s: %v", process.ID(), err)
		} else {
			logger.WriteLinef("Error occurred while awaiting resources for process %s: %v", process.ID(), err)
		}
		logger.WriteLinef("awaiting resources for process %s (resource: %s, timeout: %s): %v", process.ID(), process.AwaitResource(), process.AwaitTimeout(), err)
		appContext.AddReport(err.Error())
		appContext.RemoveRunningProcess(process)
		return err
	}

	diff := time.Since(start)
	logger.WriteLinef("Process %s starting at %v (waited %v for resource: %s)", process.ID(), time.Now(), diff, process.AwaitResource())
	return nil
}

// warnInsecureSSHTunnels prints a stderr warning for every non-skipped SSH
// tunnel unit that has insecure_ignore_host_key: true. The warning is written
// directly to os.Stderr so it is always visible regardless of log level.
func warnInsecureSSHTunnels(units map[string]*RunpUnit, skipped map[string]bool) {
	for name, unit := range units {
		if skipped[name] {
			continue
		}
		if unit.SSHTunnel != nil && unit.SSHTunnel.InsecureIgnoreHostKey {
			fmt.Fprintf(os.Stderr,
				"WARNING: insecure_ignore_host_key is enabled for unit %s — host key verification is disabled (MITM risk)\n",
				name)
		}
	}
}

func await(duration time.Duration, resources []string) error {
	ui.WriteLinef(`Awaiting resources for %s: %v resources %v`, duration, len(resources), resources)
	if len(resources) < 1 {
		ui.WriteLinef("No resources specified, waiting for duration: %s", duration)
		time.Sleep(duration)
		return nil
	}
	ui.WriteLinef("Awaiting %d resource(s) for %s: %v", len(resources), duration, resources)
	return impatient.Await(context.Background(), resources, duration)
}
