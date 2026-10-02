package core

import (
	"context"
	"errors"
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
		longest:             longestUnitName(rf),
		aborted:             make(chan struct{}),
	}
}

func longestUnitName(rf *Runpfile) int {
	ln := 0
	for _, unit := range rf.Units {
		if len(unit.Name) > ln {
			ln = len(unit.Name)
		}
	}
	return ln
}

// RunpfileExecutor Executor implementation for Runpfile.
type RunpfileExecutor struct {
	rf                  *Runpfile
	LoggerFactory       func(string, int, LoggerConfig) Logger
	longest             int
	environmentSettings *EnvironmentSettings
	newPipe             func() (*os.File, *os.File, error)
	PIDDir              string

	// running tracks every started process until it has exited, so that
	// Start does not return while children are still alive.
	running sync.WaitGroup
	// aborted is closed when a unit fails to start; units not yet started
	// are then skipped.
	aborted   chan struct{}
	abortOnce sync.Once
}

// longestName returns the longest unit name length. It is computed before any
// unit goroutine starts and never written afterwards, so reads are race-free.
func (e *RunpfileExecutor) longestName() int {
	return e.longest
}

func (e *RunpfileExecutor) initializeUnits() {
	if e.longest == 0 {
		e.longest = longestUnitName(e.rf)
	}
	if e.aborted == nil {
		e.aborted = make(chan struct{})
	}
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
		if err := resolveUnitWorkingDir(unit, e.rf.Root); err != nil {
			ui.WriteLinef("%v", err)
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
	err := e.startUnit(unit)
	// A unit with a ready condition returns from startUnit while still
	// running: wait for it so the caller does not leave it orphaned.
	e.running.Wait()
	return err
}

// abort stops the running processes of this Runpfile and prevents units not
// yet started from starting. It is called when a unit fails to start, so that
// runp does not exit leaving the units already started orphaned.
func (e *RunpfileExecutor) abort() {
	e.abortOnce.Do(func() {
		close(e.aborted)
		running := GetApplicationContext().GetRunningProcesses()
		own := make(map[string]RunpProcess)
		for _, unit := range e.rf.Units {
			p := unit.Process()
			if p == nil {
				continue
			}
			if rp, ok := running[p.ID()]; ok && rp == p {
				own[p.ID()] = p
			}
		}
		stopProcesses(own)
	})
}

func (e *RunpfileExecutor) isAborted() bool {
	select {
	case <-e.aborted:
		return true
	default:
		return false
	}
}

// errShuttingDown marks units that were not started, or were stopped,
// because another unit failed and runp is stopping everything.
var errShuttingDown = errors.New("runp is shutting down")

// unitReadiness is resolved once per unit with the outcome its dependents
// wait for: nil when the unit is ready, or the reason it never will be.
type unitReadiness struct {
	once sync.Once
	done chan struct{}
	err  error
}

func newUnitReadiness() *unitReadiness {
	return &unitReadiness{done: make(chan struct{})}
}

func (r *unitReadiness) resolve(err error) {
	r.once.Do(func() {
		r.err = err
		close(r.done)
	})
}

// Start starts all units and blocks until every started process has exited.
//
// Each unit starts as soon as all the units it depends_on are ready,
// independently of unrelated units. A unit is ready when its ready condition
// is satisfied or, for units without a ready condition, when its process has
// exited successfully. When a dependency fails (it cannot start, its ready
// condition fails or it exits with an error) its dependents are not started,
// the units already running are stopped and Start returns an error.
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

	// Validates depends_on references and cycles; the layers only give a
	// deterministic launch order, readiness is tracked per unit.
	layers, err := TopologicalLayers(e.rf.Units, skipped)
	if err != nil {
		return fmt.Errorf("cannot start: %w", err)
	}

	warnBlockingDependencies(e.rf.Units, skipped)

	states := make(map[string]*unitReadiness)
	for _, layer := range layers {
		for _, name := range layer {
			states[name] = newUnitReadiness()
		}
	}

	var mu sync.Mutex
	var errs []error
	fail := func(err error) {
		if !errors.Is(err, errShuttingDown) {
			mu.Lock()
			errs = append(errs, err)
			mu.Unlock()
		}
		e.abort()
	}

	var wg sync.WaitGroup
	for _, layer := range layers {
		for _, name := range layer {
			wg.Add(1)
			go func(u *RunpUnit, ready *unitReadiness) {
				defer wg.Done()
				// Dependents must never wait forever, whatever happens here.
				defer ready.resolve(fmt.Errorf("unit %s did not start", u.Name))
				defer func() {
					if r := recover(); r != nil {
						ui.WriteLinef("Panic in goroutine for unit %s: %v", u.Name, r)
						fail(fmt.Errorf("panic in goroutine for unit %s: %v", u.Name, r))
						GetApplicationContext().TriggerShutdown()
					}
				}()
				if err := e.awaitDependencies(u, states, skipped); err != nil {
					ready.resolve(err)
					if !errors.Is(err, errShuttingDown) {
						ui.WriteLinef("%v", err)
					}
					fail(err)
					return
				}
				if err := e.runUnit(u, ready.resolve); err != nil {
					ready.resolve(err)
					fail(err)
				}
			}(e.rf.Units[name], states[name])
		}
	}
	wg.Wait()
	e.running.Wait()

	if len(errs) > 0 {
		return fmt.Errorf("%d unit(s) failed to start", len(errs))
	}
	return nil
}

// warnBlockingDependencies notes the dependencies without a ready condition:
// their dependents start only after they exit successfully, which is never
// for a long-running service.
func warnBlockingDependencies(units map[string]*RunpUnit, skipped map[string]bool) {
	names := make([]string, 0, len(units))
	for name := range units {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if skipped[name] {
			continue
		}
		for _, dep := range units[name].DependsOn {
			if d, ok := units[dep]; ok && !skipped[dep] && !d.Ready.IsSet() {
				ui.WriteLinef("Unit %s starts after %s exits successfully (%s has no ready condition)", name, dep, dep)
			}
		}
	}
}

// awaitDependencies blocks until every dependency of u is ready. It returns an
// error if a dependency failed or runp is stopping.
func (e *RunpfileExecutor) awaitDependencies(u *RunpUnit, states map[string]*unitReadiness, skipped map[string]bool) error {
	for _, dep := range u.DependsOn {
		if skipped[dep] {
			continue
		}
		st := states[dep]
		select {
		case <-st.done:
			if st.err != nil {
				return fmt.Errorf("unit %s not started: dependency %s failed: %v", u.Name, dep, st.err)
			}
		case <-e.aborted:
			return fmt.Errorf("unit %s not started: %w", u.Name, errShuttingDown)
		}
	}
	return nil
}

// startUnit starts unit and returns once it is ready (see runUnit).
func (e *RunpfileExecutor) startUnit(unit *RunpUnit) error {
	return e.runUnit(unit, func(error) {})
}

// runUnit starts the unit's process. ready is called once the outcome
// relevant to dependents is known:
//   - units with a ready condition: when the condition is satisfied or fails;
//     runUnit then returns while the process keeps running.
//   - other units: when the process exits (with its exit error); runUnit
//     returns after the exit.
//
// The returned error reports failures to start the process; ready is not
// called in that case.
func (e *RunpfileExecutor) runUnit(unit *RunpUnit, ready func(error)) error {
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

	if e.isAborted() {
		appContext.RemoveRunningProcess(process)
		return fmt.Errorf("unit %s not started: %w", unit.Name, errShuttingDown)
	}

	r, w, err := e.newPipe()
	if err != nil {
		appContext.RemoveRunningProcess(process)
		return fmt.Errorf("os.Pipe: %w", err)
	}
	cmd.Stdout(w)
	cmd.Stderr(w)

	var pwg sync.WaitGroup
	pwg.Add(1)

	if err := e.startProcessCommand(cmd, unit, process, logger, appContext, w, &pwg); err != nil {
		r.Close()
		return err
	}

	w.Close()
	e.running.Add(1)
	// exitErr is written by the monitor goroutine before pwg.Done and read
	// only after pwg.Wait.
	var exitErr error
	e.monitorProcessExit(cmd, process, logger, appContext, &pwg, &exitErr)
	go func() {
		pwg.Wait()
		e.running.Done()
	}()
	if e.isAborted() {
		// abort ran between the check above and the start: stop this unit too.
		stopRunningProcess(process)
		e.readProcessOutput(r, process, logger)
		return fmt.Errorf("unit %s stopped: %w", unit.Name, errShuttingDown)
	}

	rc := unit.Ready
	if rc.IsSet() {
		// Output is read in a background goroutine so that AwaitReady can
		// scan it for the ready pattern. runUnit returns as soon as readiness
		// is known; the process keeps running and is tracked by e.running.
		lineCh := make(chan string, 128)
		go e.readProcessOutputToChannel(r, process, logger, lineCh)
		if err := AwaitReady(rc, lineCh, logger); err != nil {
			logger.WriteLinef("Readiness check failed for unit %s: %v", unit.Name, err)
			ready(fmt.Errorf("readiness check failed: %w", err))
			return nil
		}
		ready(nil)
		return nil
	}

	e.readProcessOutput(r, process, logger)
	pwg.Wait()
	if exitErr != nil {
		ready(fmt.Errorf("process exited with error: %w", exitErr))
	} else {
		ready(nil)
	}
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
