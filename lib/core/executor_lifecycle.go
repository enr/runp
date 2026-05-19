package core

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

func (e *RunpfileExecutor) monitorProcessExit(cmd RunpCommand, process RunpProcess, logger Logger, appContext *ApplicationContext, pwg *sync.WaitGroup) {
	exit := make(chan error, 1)
	go func() {
		exit <- cmd.Wait()
		logger.WriteLinef("Process %s finished: %s", process.ID(), cmd)
	}()

	go func() {
		defer pwg.Done()
		defer appContext.RemoveRunningProcess(process)
		defer process.PostStop()

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

	logger.WriteLinef("Error occurred while running process %s: running process %s: %v", process.ID(), process.ID(), err)

	var b bytes.Buffer
	fmt.Fprintf(&b, "Error type %T occurred in process %s: %s", err, process.ID(), err.Error())
	logger.Write(b.Bytes())
	appContext.AddReport(err.Error())
}
