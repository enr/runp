package core

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/bww/impatient"
	"github.com/pkg/errors"
)

// NewExecutor creates new RunpfileExecutor
func NewExecutor(rf *Runpfile) *RunpfileExecutor {
	return &RunpfileExecutor{
		rf:                  rf,
		LoggerFactory:       createProcessLogger,
		environmentSettings: loadEnvironmentSettings(),
		newPipe:             os.Pipe,
	}
}

// RunpfileExecutor Executor implementation for Runpfile.
type RunpfileExecutor struct {
	rf                  *Runpfile
	LoggerFactory       func(string, int, LoggerConfig) Logger
	longest             int
	environmentSettings *EnvironmentSettings
	newPipe             func() (*os.File, *os.File, error)
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

// Start call start on all processes.
func (e *RunpfileExecutor) Start() error {
	e.initializeUnits()
	skipped := e.skippedUnits()
	if len(skipped) > 0 {
		names := make([]string, 0, len(skipped))
		for name := range skipped {
			names = append(names, name)
		}
		ui.WriteLinef("Units skipped due to unsatisfied preconditions: %v", names)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	for _, unit := range e.rf.Units {
		if skipped[unit.Name] {
			ui.WriteLinef("Skipping unit: %s", unit.Name)
			continue
		}
		wg.Add(1)
		go func(u *RunpUnit) {
			defer wg.Done()
			if err := e.startUnit(u); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}(unit)
	}

	wg.Wait()

	if len(errs) > 0 {
		return fmt.Errorf("%d unit(s) failed to start", len(errs))
	}
	return nil
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
		ctx := fmt.Sprintf("awaiting resources for process %s (resource: %s, timeout: %s)", process.ID(), process.AwaitResource(), process.AwaitTimeout())
		logger.WriteLinef("%+v", errors.Wrap(err, ctx))
		appContext.AddReport(err.Error())
		appContext.RemoveRunningProcess(process)
		return err
	}

	diff := time.Since(start)
	logger.WriteLinef("Process %s starting at %v (waited %v for resource: %s)", process.ID(), time.Now(), diff, process.AwaitResource())
	return nil
}

func (e *RunpfileExecutor) verifyProcessStartability(process RunpProcess, logger Logger, appContext *ApplicationContext) error {
	startable, err := process.IsStartable()
	if err != nil {
		logger.WriteLinef("Failed to verify startability for process %s: %+v", process.ID(), errors.Wrap(err, "startability check"))
		appContext.RemoveRunningProcess(process)
		return err
	}
	if !startable {
		logger.WriteLinef("Process %s cannot be started", process.ID())
		appContext.RemoveRunningProcess(process)
		return fmt.Errorf("process %s cannot be started", process.ID())
	}
	return nil
}

func (e *RunpfileExecutor) startProcessCommand(cmd RunpCommand, unit *RunpUnit, process RunpProcess, logger Logger, appContext *ApplicationContext, w *os.File, pwg *sync.WaitGroup) error {
	err := cmd.Start()
	if err != nil {
		w.Close()
		ctx := fmt.Sprintf("starting process %s", unit.Name)
		logger.WriteLinef("Failed to start process %s: %+v", unit.Name, errors.Wrap(err, ctx))
		appContext.RemoveRunningProcess(process)
		pwg.Done()
		return err
	}
	logger.Debugf("Process %s started successfully", process.ID())
	return nil
}

func (e *RunpfileExecutor) monitorProcessExit(cmd RunpCommand, process RunpProcess, logger Logger, appContext *ApplicationContext, pwg *sync.WaitGroup) {
	exit := make(chan error, 1)
	go func() {
		exit <- cmd.Wait()
		logger.WriteLinef("Process %s finished: %s", process.ID(), cmd)
	}()

	go func() {
		defer pwg.Done()
		defer appContext.RemoveRunningProcess(process)

		err := <-exit
		if err != nil {
			if e.isGracefulShutdown(err, process, logger) {
				return
			}
			e.handleProcessError(err, process, logger, appContext)
		} else {
			logger.WriteLinef("Process %s completed successfully", process.ID())
		}
	}()
}

func (e *RunpfileExecutor) isGracefulShutdown(err error, process RunpProcess, logger Logger) bool {
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return false
	}

	errMsg := err.Error()
	// Check for common graceful shutdown error messages
	if errMsg == "signal: terminated" || errMsg == "signal: interrupt" || errMsg == "signal: killed" {
		logger.Debugf("Process %s terminated by signal (graceful shutdown): %s", process.ID(), errMsg)
		return true
	}

	exitCode := exitErr.ExitCode()
	// Check for common graceful shutdown exit codes on Unix systems
	if exitCode == 128+int(syscall.SIGTERM) || exitCode == 128+int(syscall.SIGINT) || exitCode == 128+int(syscall.SIGKILL) {
		logger.Debugf("Process %s terminated by signal (graceful shutdown), exit code: %d", process.ID(), exitCode)
		return true
	}

	// On Windows, when a process is killed with Kill(), it may generate exit code 1
	// but we cannot assume all exit code 1 are graceful shutdowns.
	// Verify if the application is shutting down.
	// If shutting down, consider all *exec.ExitError as graceful shutdown.
	appContext := GetApplicationContext()
	if appContext.IsShuttingDown() {
		// Application is shutting down, so this is likely a graceful shutdown
		logger.Debugf("Process %s terminated during application shutdown (graceful shutdown), exit code: %d", process.ID(), exitCode)
		return true
	}

	return false
}

func (e *RunpfileExecutor) handleProcessError(err error, process RunpProcess, logger Logger, appContext *ApplicationContext) {
	switch err.(type) {
	case *os.SyscallError:
		logger.WriteLinef("System call error in process %s: %s", process.ID(), err.Error())
	default:
		logger.WriteLinef("Unexpected error type in process %s: %T", process.ID(), err)
	}

	ctx := fmt.Sprintf("running process %s", process.ID())
	logger.WriteLinef("Error occurred while running process %s: %+v", process.ID(), errors.Wrap(err, ctx))

	var b bytes.Buffer
	fmt.Fprintf(&b, "Error type %T occurred in process %s: %s", err, process.ID(), err.Error())
	logger.Write(b.Bytes())
	appContext.AddReport(err.Error())
}

func (e *RunpfileExecutor) readProcessOutput(r *os.File, process RunpProcess, logger Logger) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		logger.Write(scanner.Bytes())
	}
	if err := scanner.Err(); err != nil {
		logger.WriteLinef("Failed to read output from process %s: %v", process.ID(), err)
	}
}
