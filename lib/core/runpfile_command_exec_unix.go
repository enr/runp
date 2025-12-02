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

	// Send SIGTERM for graceful shutdown (allows bash trap functions to execute)
	err := p.Signal(syscall.SIGTERM)
	if err != nil {
		// Process might have already exited
		return nil
	}

	// Wait for the process to exit gracefully
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(timeout):
		// Timeout reached, force kill with SIGKILL
		if id != "" {
			ui.Debugf("Process %s did not terminate gracefully within %v, forcing kill", id, timeout)
		}
		return p.Kill()
	case err := <-done:
		// Process exited gracefully
		return err
	}
}
