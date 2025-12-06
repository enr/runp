//go:build windows
// +build windows

package core

import (
	"os/exec"
	"syscall"
	"time"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procOpenProcess  = kernel32.NewProc("OpenProcess")
	procCloseHandle  = kernel32.NewProc("CloseHandle")
	procGetLastError = kernel32.NewProc("GetLastError")
)

const (
	PROCESS_QUERY_INFORMATION = 0x0400
	PROCESS_TERMINATE         = 0x0001
	ERROR_INVALID_PARAMETER   = 87
	ERROR_ACCESS_DENIED       = 5
)

// stopWithGracefulShutdown implements graceful shutdown for Windows:
// On Windows, SIGTERM is not available, so we use Kill() directly.
// For bash scripts running in Git Bash or WSL, Kill() will still allow
// the process to handle termination gracefully.
func stopWithGracefulShutdown(cmd *exec.Cmd, timeout time.Duration) error {
	return stopWithGracefulShutdownWithID(cmd, timeout, "")
}

// processStatus represents the status of a process check
type processStatus int

const (
	processExited       processStatus = iota // Process has exited
	processRunning                           // Process is running
	processAccessDenied                      // Process exists but access is denied
)

// checkProcessStatus checks the status of a process on Windows
// It distinguishes between "process doesn't exist" and "access denied"
func checkProcessStatus(pid int) processStatus {
	handle, _, _ := procOpenProcess.Call(PROCESS_QUERY_INFORMATION, 0, uintptr(pid))
	if handle != 0 {
		// Successfully opened the process - it exists and we have access
		procCloseHandle.Call(handle)
		return processRunning
	}

	// OpenProcess failed - need to check the error code
	// GetLastError returns the last error code set by Windows API
	lastError, _, _ := procGetLastError.Call()
	errorCode := int(lastError)

	switch errorCode {
	case ERROR_INVALID_PARAMETER:
		// Process doesn't exist (invalid PID)
		return processExited
	case ERROR_ACCESS_DENIED:
		// Process exists but we don't have permission to access it
		// This means the process is likely still running
		return processAccessDenied
	default:
		// For other errors, assume process doesn't exist
		// This is safer than assuming it's running
		return processExited
	}
}

// isProcessRunning checks if a process is still running on Windows
// Returns true only if we can confirm the process is running.
// Returns false if the process has exited or if we can't determine its status.
func isProcessRunning(pid int) bool {
	status := checkProcessStatus(pid)
	return status == processRunning
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

	// Check if process is still running before attempting to kill it
	if !isProcessRunning(p.Pid) {
		// Process has already exited
		return nil
	}

	// On Windows, SIGTERM doesn't exist. We use Kill() which sends a termination signal.
	// For bash scripts in Git Bash/WSL, this should still allow graceful handling.
	err := p.Kill()
	if err != nil {
		// If Kill() fails, check the process status to determine if it actually exited
		status := checkProcessStatus(p.Pid)
		switch status {
		case processExited:
			// Process has exited - we can ignore the Kill() error
			return nil
		case processAccessDenied:
			// Process exists but we don't have access - it's likely still running
			// Return the original error to indicate failure to terminate
			return err
		case processRunning:
			// Process is confirmed running but Kill() failed - return the error
			return err
		default:
			// Unknown status - return the error to be safe
			return err
		}
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
		// Check if process is still running before attempting to kill it again
		if !isProcessRunning(p.Pid) {
			return nil
		}
		err := p.Kill()
		if err != nil {
			// If Kill() fails, check the process status to determine if it actually exited
			status := checkProcessStatus(p.Pid)
			switch status {
			case processExited:
				// Process has exited - we can ignore the Kill() error
				return nil
			case processAccessDenied:
				// Process exists but we don't have access - it's likely still running
				// Return the original error to indicate failure to terminate
				return err
			case processRunning:
				// Process is confirmed running but Kill() failed - return the error
				return err
			default:
				// Unknown status - return the error to be safe
				return err
			}
		}
	}
	return nil
}
