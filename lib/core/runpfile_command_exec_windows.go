//go:build windows
// +build windows

package core

import (
	"os/exec"
	"time"
)

// stopWithGracefulShutdown implements graceful shutdown for Windows:
// On Windows, SIGTERM is not available, so we use Kill() directly.
// For bash scripts running in Git Bash or WSL, Kill() will still allow
// the process to handle termination gracefully.
func stopWithGracefulShutdown(cmd *exec.Cmd, timeout time.Duration) error {
	return stopWithGracefulShutdownWithID(cmd, timeout, "")
}

// stopWithGracefulShutdownWithID implements graceful shutdown for Windows with process ID logging
func stopWithGracefulShutdownWithID(cmd *exec.Cmd, timeout time.Duration, id string) error {
	p := cmd.Process
	if p == nil {
		return nil
	}
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		return nil
	}

	// On Windows, SIGTERM doesn't exist. We use Kill() which sends a termination signal.
	// For bash scripts in Git Bash/WSL, this should still allow graceful handling.
	err := p.Kill()
	if err != nil {
		// Process might have already exited
		return nil
	}

	// Poll process state instead of calling Wait() to avoid conflicts
	// The executor's main goroutine will handle Wait() when the process exits
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		<-ticker.C
		// Check if process has exited by polling ProcessState
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			// Process exited
			return nil
		}
	}

	// On Windows, Kill() should terminate immediately, but log if it doesn't
	if id != "" {
		ui.Debugf("Process %s did not terminate within %v on Windows", id, timeout)
	}
	// Try killing again if still running
	if cmd.ProcessState == nil || !cmd.ProcessState.Exited() {
		return p.Kill()
	}
	return nil
}
