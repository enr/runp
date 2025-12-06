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

	status := checkProcessStatus(p.Pid)
	if status == processExited {
		return nil
	}
	// On Windows, SIGTERM doesn't exist. We use Kill() which sends a termination signal.
	// For bash scripts in Git Bash/WSL, this should still allow graceful handling.
	err := p.Kill()
	if err != nil {
		// If Kill() fails, check the process status to determine if it actually exited
		status = checkProcessStatus(p.Pid)
		switch status {
		case processExited:
			// Process has exited - we can ignore the Kill() error
			return nil
		case processAccessDenied:
			// Process exists but we don't have access - try TerminateProcess directly
			// This can work in cases where p.Kill() fails due to permission issues
			termErr := terminateProcessDirectly(p.Pid)
			if termErr == nil {
				// TerminateProcess succeeded - continue with polling
				// Don't return here, let the polling loop handle it
			} else {
				// TerminateProcess also failed - wait a bit and check if process exited
				// Sometimes the process might exit on its own or be terminated by the system
				time.Sleep(100 * time.Millisecond)
				if !isProcessRunning(p.Pid) {
					// Process has exited - consider it successful
					return nil
				}
				// Process is still running and we can't terminate it
				// Log a warning but don't fail - we've done our best
				if id != "" {
					ui.Debugf("Process %s (PID %d) could not be terminated: %v (access denied)", id, p.Pid, err)
				}
				// Return nil to avoid failing the stop operation
				// The process might be terminated by the system or exit on its own
				return nil
			}
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
				// Process exists but we don't have access - try TerminateProcess directly
				termErr := terminateProcessDirectly(p.Pid)
				if termErr == nil {
					// TerminateProcess succeeded
					return nil
				}
				// TerminateProcess also failed - wait a bit and check if process exited
				time.Sleep(100 * time.Millisecond)
				if !isProcessRunning(p.Pid) {
					// Process has exited - consider it successful
					return nil
				}
				// Process is still running and we can't terminate it
				// Log a warning but don't fail - we've done our best
				if id != "" {
					ui.Debugf("Process %s (PID %d) could not be terminated after timeout: %v (access denied)", id, p.Pid, err)
				}
				// Return nil to avoid failing the stop operation
				return nil
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
