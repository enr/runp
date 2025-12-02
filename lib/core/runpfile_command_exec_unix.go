//go:build darwin || freebsd || linux || netbsd || openbsd
// +build darwin freebsd linux netbsd openbsd

package core

import (
	"os/exec"
	"syscall"
	"time"
)

// stopWithGracefulShutdown implements graceful shutdown for Unix systems:
// sends SIGTERM first (allows bash trap functions to execute), waits for timeout, then SIGKILL
func stopWithGracefulShutdown(cmd *exec.Cmd, timeout time.Duration) error {
	return stopWithGracefulShutdownWithID(cmd, timeout, "")
}

// stopWithGracefulShutdownWithID implements graceful shutdown for Unix systems with process ID logging
func stopWithGracefulShutdownWithID(cmd *exec.Cmd, timeout time.Duration, id string) error {
	p := cmd.Process
	if p == nil {
		return nil
	}
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		return nil
	}

	// Send SIGTERM to the process group for graceful shutdown
	// This ensures bash and its child processes receive the signal
	// Using negative PID sends signal to the entire process group
	pid := p.Pid
	pgid, err := syscall.Getpgid(pid)
	if err == nil {
		// Send SIGTERM to the entire process group
		err = syscall.Kill(-pgid, syscall.SIGTERM)
		if err != nil {
			// Fallback: try sending to the process directly
			err = p.Signal(syscall.SIGTERM)
			if err != nil {
				// Process might have already exited
				return nil
			}
		}
	} else {
		// Fallback: send to process directly if we can't get PGID
		err = p.Signal(syscall.SIGTERM)
		if err != nil {
			// Process might have already exited
			return nil
		}
	}

	// Poll process state instead of calling Wait() to avoid "waitid: no child processes" error
	// The executor's main goroutine will handle Wait() when the process exits
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		<-ticker.C
		// Check if process has exited by polling ProcessState
		// Note: We can't call Wait() here as it's already being called by the executor
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			// Process exited gracefully
			return nil
		}
		// Check if process is still alive by sending signal 0 (no-op)
		// If the process has exited, Signal(0) will return an error
		if err := p.Signal(syscall.Signal(0)); err != nil {
			// Process no longer exists (exited)
			return nil
		}
	}

	// Timeout reached, force kill with SIGKILL
	if id != "" {
		ui.Debugf("Process %s did not terminate gracefully within %v, forcing kill", id, timeout)
	}
	// Send SIGKILL to the process group
	killPid := p.Pid
	killPgid, killErr := syscall.Getpgid(killPid)
	if killErr == nil {
		// Send SIGKILL to the entire process group
		killErr = syscall.Kill(-killPgid, syscall.SIGKILL)
		if killErr != nil {
			// Fallback: try killing the process directly
			return p.Kill()
		}
		return killErr
	}
	// Fallback: kill process directly
	return p.Kill()
}
