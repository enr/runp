//go:build windows
// +build windows

package core

import (
	"os"
	"syscall"
	"time"
)

const (
	PROCESS_QUERY_INFORMATION = 0x0400
	PROCESS_TERMINATE         = 0x0001
	ERROR_INVALID_PARAMETER   = 87
	ERROR_ACCESS_DENIED       = 5
)

// processStatus represents the status of a process check
type processStatus int

const (
	processExited       processStatus = iota // Process has exited
	processRunning                           // Process is running
	processAccessDenied                      // Process exists but access is denied
)

// checkProcessStatus checks the status of a process on Windows
// It distinguishes between "process doesn't exist" and "access denied".
//
// The error comes from the OpenProcess call itself: the last Win32 error is
// per thread, so reading it with a separate GetLastError call (the goroutine
// may have moved to another OS thread in between) gave unreliable results.
func checkProcessStatus(pid int) processStatus {
	handle, err := syscall.OpenProcess(PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err == nil {
		// Successfully opened the process - it exists and we have access
		syscall.CloseHandle(handle)
		return processRunning
	}

	switch err {
	case syscall.Errno(ERROR_INVALID_PARAMETER):
		// Process doesn't exist (invalid PID)
		return processExited
	case syscall.ERROR_ACCESS_DENIED:
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
	handle, err := syscall.OpenProcess(PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return err
	}
	defer syscall.CloseHandle(handle)

	// Terminate the process with exit code 1
	return syscall.TerminateProcess(handle, 1)
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
func pollProcessUntilExit(exited func() bool, timeout time.Duration) bool {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		<-ticker.C
		// Check if the executor has reaped the process
		if exited() {
			// Process exited
			return true
		}
	}
	return false
}

// attemptFinalKill attempts to kill the process after timeout, handling errors appropriately
func attemptFinalKill(p *os.Process, pid int, id string) error {
	// Check if process is still running before attempting to kill it again
	if !isProcessRunning(pid) {
		return nil
	}
	err := killProcessTree(p)
	if err != nil {
		return handleKillError(pid, err, id, true)
	}
	return nil
}

// stopProcess implements shutdown for Windows: SIGTERM is not available, so
// the process tree is terminated directly. exited reports whether the executor
// has already reaped the process.
func stopProcess(p *os.Process, exited func() bool, timeout time.Duration, id string) error {
	if p == nil || exited() {
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

	// On Windows, SIGTERM doesn't exist: the process is terminated. The whole
	// tree is terminated, since Kill() alone (TerminateProcess) would stop the
	// "cmd /C" shell and leave its children running.
	err := killProcessTree(p)
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
	if pollProcessUntilExit(exited, timeout) {
		return nil
	}

	// On Windows, Kill() should terminate immediately, but log if it doesn't
	if id != "" {
		ui.Debugf("Process %s did not terminate within %v on Windows", id, timeout)
	}

	// Try killing again if still running
	if !exited() {
		return attemptFinalKill(p, p.Pid, id)
	}

	return nil
}
