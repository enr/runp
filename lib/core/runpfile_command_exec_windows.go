//go:build windows
// +build windows

package core

import (
	"os/exec"
	"syscall"
	"time"
)

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procOpenProcess      = kernel32.NewProc("OpenProcess")
	procCloseHandle      = kernel32.NewProc("CloseHandle")
	procGetLastError     = kernel32.NewProc("GetLastError")
	procTerminateProcess = kernel32.NewProc("TerminateProcess")
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

// terminateProcessDirectly attempts to terminate a process using TerminateProcess directly
// This can work in cases where p.Kill() fails due to permission issues
func terminateProcessDirectly(pid int) error {
	// Try to open the process with TERMINATE permission
	handle, _, _ := procOpenProcess.Call(PROCESS_TERMINATE, 0, uintptr(pid))
	if handle == 0 {
		// Failed to open process - return error
		lastError, _, _ := procGetLastError.Call()
		return syscall.Errno(lastError)
	}
	defer procCloseHandle.Call(handle)

	// Terminate the process with exit code 1
	result, _, _ := procTerminateProcess.Call(handle, 1)
	if result == 0 {
		// TerminateProcess failed
		lastError, _, _ := procGetLastError.Call()
		return syscall.Errno(lastError)
	}

	return nil
}

// handleKillError handles errors when Kill() fails, attempting alternative termination methods
func handleKillError(pid int, killErr error, id string, afterTimeout bool) error {
	status := checkProcessStatus(pid)
	switch status {
	case processExited:
		// Process has exited - we can ignore the Kill() error
		return nil
	case processAccessDenied:
		// Process exists but we don't have access - try TerminateProcess directly
		termErr := terminateProcessDirectly(pid)
		if termErr == nil {
			// TerminateProcess succeeded
			return nil
		}
		// TerminateProcess also failed - wait a bit and check if process exited
		time.Sleep(100 * time.Millisecond)
		if !isProcessRunning(pid) {
			// Process has exited - consider it successful
			return nil
		}
		// Process is still running and we can't terminate it
		// Log a warning but don't fail - we've done our best
		if id != "" {
			if afterTimeout {
				ui.Debugf("Process %s (PID %d) could not be terminated after timeout: %v (access denied)", id, pid, killErr)
			} else {
				ui.Debugf("Process %s (PID %d) could not be terminated: %v (access denied)", id, pid, killErr)
			}
		}
		// Return nil to avoid failing the stop operation
		return nil
	case processRunning:
		// Process is confirmed running but Kill() failed - return the error
		return killErr
	default:
		// Unknown status - return the error to be safe
		return killErr
	}
}

// pollProcessUntilExit polls the process state until it exits or the timeout expires
func pollProcessUntilExit(cmd *exec.Cmd, timeout time.Duration) bool {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		<-ticker.C
		// Check if process has exited by polling ProcessState
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			// Process exited
			return true
		}
	}
	return false
}

// attemptFinalKill attempts to kill the process after timeout, handling errors appropriately
func attemptFinalKill(cmd *exec.Cmd, pid int, id string) error {
	// Check if process is still running before attempting to kill it again
	if !isProcessRunning(pid) {
		return nil
	}
	err := cmd.Process.Kill()
	if err != nil {
		return handleKillError(pid, err, id, true)
	}
	return nil
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
		return nil
	}

	status := checkProcessStatus(p.Pid)
	if status == processExited {
		return nil
	}

	// On Windows, SIGTERM doesn't exist. We use Kill() which sends a termination signal.
	// For bash scripts in Git Bash/WSL, this should still allow graceful handling.
	err := p.Kill()
	if err != nil {
		// Handle Kill() error - may continue with polling if TerminateProcess succeeds
		handleErr := handleKillError(p.Pid, err, id, false)
		if handleErr != nil {
			return handleErr
		}
		// If handleKillError returned nil, it means either:
		// - Process exited
		// - TerminateProcess succeeded (continue with polling)
		// - Access denied but we'll let polling handle it
	}

	// Poll process state instead of calling Wait() to avoid conflicts
	// The executor's main goroutine will handle Wait() when the process exits
	if pollProcessUntilExit(cmd, timeout) {
		return nil
	}

	// On Windows, Kill() should terminate immediately, but log if it doesn't
	if id != "" {
		ui.Debugf("Process %s did not terminate within %v on Windows", id, timeout)
	}

	// Try killing again if still running
	if cmd.ProcessState == nil || !cmd.ProcessState.Exited() {
		return attemptFinalKill(cmd, p.Pid, id)
	}

	return nil
}
