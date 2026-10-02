package core

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"sync"
)

// monitorProcessExit waits for cmd in the background. The exit error is
// stored in *exitErr (if not nil) before pwg is released.
func (e *RunpfileExecutor) monitorProcessExit(cmd RunpCommand, process RunpProcess, logger Logger, appContext *ApplicationContext, pwg *sync.WaitGroup, exitErr *error) {
	exit := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				ui.WriteLinef("Panic waiting for process %s to finish: %v", process.ID(), r)
				exit <- fmt.Errorf("panic waiting for process to finish: %v", r)
			}
		}()
		exit <- cmd.Wait()
		logger.WriteLinef("Process %s finished: %s", process.ID(), cmd)
	}()

	go func() {
		defer pwg.Done()
		defer appContext.RemoveRunningProcess(process)
		defer process.PostStop()
		defer func() {
			if r := recover(); r != nil {
				ui.WriteLinef("Panic in lifecycle handler for process %s: %v", process.ID(), r)
				appContext.AddReport(fmt.Sprintf("panic in lifecycle handler for process %s: %v", process.ID(), r))
				appContext.TriggerShutdown()
			}
		}()

		err := <-exit
		if exitErr != nil {
			*exitErr = err
		}
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

func (e *RunpfileExecutor) verifyProcessStartability(process RunpProcess, logger Logger, appContext *ApplicationContext) error {
	startable, err := process.IsStartable()
	if err != nil {
		logger.WriteLinef("Failed to verify startability for process %s: %v (startability check)", process.ID(), err)
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

// isGracefulShutdown reports whether a process exit is the expected effect of
// runp stopping it: runp is shutting down (Ctrl-C, or a unit failed to start)
// and the process exited, typically by signal. Outside a shutdown a process
// killed by a signal (e.g. by the OOM killer) is an error and is reported.
func (e *RunpfileExecutor) isGracefulShutdown(err error, process RunpProcess, logger Logger) bool {
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return false
	}
	if !GetApplicationContext().IsShuttingDown() && !e.isAborted() {
		return false
	}
	// On Windows a killed process may exit with code 1: during a shutdown
	// every *exec.ExitError is considered graceful.
	logger.Debugf("Process %s terminated during shutdown (graceful shutdown): %s (exit code %d)", process.ID(), err, exitErr.ExitCode())
	return true
}

func (e *RunpfileExecutor) handleProcessError(err error, process RunpProcess, logger Logger, appContext *ApplicationContext) {
	switch err.(type) {
	case *os.SyscallError:
		logger.WriteLinef("System call error in process %s: %s", process.ID(), err.Error())
	default:
		logger.WriteLinef("Unexpected error type in process %s: %T", process.ID(), err)
	}

	logger.WriteLinef("Error occurred while running process %s: running process %s: %v", process.ID(), process.ID(), err)

	var b bytes.Buffer
	fmt.Fprintf(&b, "Error type %T occurred in process %s: %s", err, process.ID(), err.Error())
	logger.Write(b.Bytes())
	appContext.AddReport(err.Error())
}
