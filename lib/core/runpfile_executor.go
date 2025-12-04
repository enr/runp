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
	}
}

// RunpfileExecutor Executor implementation for Runpfile.
type RunpfileExecutor struct {
	rf                  *Runpfile
	LoggerFactory       func(string, int, LoggerConfig) Logger
	longest             int
	environmentSettings *EnvironmentSettings
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
func (e *RunpfileExecutor) Start() {

	var wg sync.WaitGroup

	var host *HostProcess
	var container *ContainerProcess
	var sshTunnel *SSHTunnelProcess
	for _, unit := range e.rf.Units {
		unit.vars = e.rf.Vars
		unit.secretKey = e.rf.SecretKey
		unit.environmentSettings = e.environmentSettings
		// Reset process to force rebuild with updated vars
		unit.process = nil
		if unit.Host != nil {
			host = unit.Host
			host.vars = unit.vars
			host.secretKey = unit.secretKey
			host.stopTimeout = unit.StopTimeout
			host.environmentSettings = e.environmentSettings
		}
		if unit.Container != nil {
			container = unit.Container
			container.vars = unit.vars
			container.secretKey = unit.secretKey
			container.stopTimeout = unit.StopTimeout
			container.environmentSettings = e.environmentSettings
		}
		if unit.SSHTunnel != nil {
			sshTunnel = unit.SSHTunnel
			sshTunnel.vars = unit.vars
			sshTunnel.secretKey = unit.secretKey
			sshTunnel.stopTimeout = unit.StopTimeout
			sshTunnel.environmentSettings = e.environmentSettings
		}
	}

	skipped := []string{}
	var pr PreconditionVerifyResult
	for _, unit := range e.rf.Units {
		if unit.Host != nil {
			host = unit.Host
			pr = host.VerifyPreconditions()
			if pr.Vote != Proceed {
				skipped = append(skipped, unit.Name)
				ui.WriteLinef("Preconditions not satisfied for unit %s (%s): %v", unit.Name, pr.Vote, pr.Reasons)
				continue
			}
		}
		if unit.Container != nil {
			container = unit.Container
			pr = container.VerifyPreconditions()
			if pr.Vote != Proceed {
				skipped = append(skipped, unit.Name)
				ui.WriteLinef("Preconditions not satisfied for unit %s (%s): %v", unit.Name, pr.Vote, pr.Reasons)
				continue
			}
		}
		if unit.SSHTunnel != nil {
			sshTunnel = unit.SSHTunnel
			pr = sshTunnel.VerifyPreconditions()
			if pr.Vote != Proceed {
				skipped = append(skipped, unit.Name)
				ui.WriteLinef("Preconditions not satisfied for unit %s (%s): %v", unit.Name, pr.Vote, pr.Reasons)
				continue
			}
		}
	}

	if len(skipped) > 0 {
		ui.WriteLinef("Units skipped due to unsatisfied preconditions: %v", skipped)
	}

	for _, unit := range e.rf.Units {
		if sliceContains(skipped, unit.Name) {
			ui.WriteLinef("Skipping unit: %s", unit.Name)
			continue
		}
		wg.Add(1)
		go e.startUnit(unit, &wg)
	}

	wg.Wait()
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

func (e *RunpfileExecutor) startUnit(unit *RunpUnit, wg *sync.WaitGroup) {
	defer wg.Done()

	logger := e.LoggerFactory(unit.Name, e.longestName(), processLoggerConfiguration)
	process := unit.Process()
	logger.WriteLinef("Starting unit %s (working directory: %s)", unit.Name, process.Dir())

	appContext := GetApplicationContext()
	appContext.RegisterRunningProcess(process)

	cmd, err := e.setupProcessCommand(unit, process, logger, appContext)
	if err != nil {
		return
	}

	if err := e.handleAwaitResources(process, logger, appContext); err != nil {
		return
	}

	logger.Debugf("Command for process %s: %v", process.ID(), cmd)

	if err := e.verifyProcessStartability(process, logger, appContext); err != nil {
		return
	}

	r, w, _ := os.Pipe()
	cmd.Stdout(w)
	cmd.Stderr(w)

	var pwg sync.WaitGroup
	pwg.Add(1)

	if err := e.startProcessCommand(cmd, unit, process, logger, appContext, w, &pwg); err != nil {
		return
	}

	w.Close()
	e.monitorProcessExit(cmd, process, logger, appContext, &pwg)
	e.readProcessOutput(r, process, logger)
	pwg.Wait()
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
	exit := make(chan error, 2)
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
	// Verify if this is a graceful shutdown before logging as error
	if e.isGracefulShutdown(err, process, logger) {
		// Graceful shutdown: do not log as error, only at debug level if needed
		logger.Debugf("Process %s terminated gracefully, skipping error reporting", process.ID())
		return
	}

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
