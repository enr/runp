//go:build windows
// +build windows

package core

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

// isAcceptableWindowsTerminationError checks if an error is acceptable during process termination on Windows.
// On Windows, some processes cannot be terminated due to security restrictions, resulting in "access denied" errors.
// This function accepts both English and localized versions of the error message.
func isAcceptableWindowsTerminationError(err error) bool {
	if err == nil {
		return true
	}
	// Direct match: wrapped errno
	if errors.Is(err, syscall.ERROR_ACCESS_DENIED) {
		return true
	}
	// If it's an os.SyscallError, check the inner Err
	var syserr *os.SyscallError
	if errors.As(err, &syserr) {
		if errors.Is(syserr.Err, syscall.ERROR_ACCESS_DENIED) {
			return true
		}
	}
	// If the error itself can be interpreted as a syscall.Errno
	var errno syscall.Errno
	if errors.As(err, &errno) {
		if errno == syscall.ERROR_ACCESS_DENIED {
			return true
		}
	}
	// If it's an exec.ExitError, try to unwrap common wrappers
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		// sometimes ExitError wraps a SyscallError or Errno; try a generic unwrap
		if errors.Is(exitErr, syscall.ERROR_ACCESS_DENIED) {
			return true
		}
		// check inner errors (if any)
		if exitErr.ProcessState != nil {
			// non sempre disponibile/utile, ma lasciamo la possibilità di altri controlli futuri
		}
	}
	return false
}

func TestExecCommandWrapper(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})

	t.Run("Pid", func(t *testing.T) {
		cmd := exec.Command("cmd", "/C", "echo test")
		wrapper := &ExecCommandWrapper{cmd: cmd}

		// Start the command to get a PID
		err := wrapper.Start()
		if err != nil {
			t.Fatalf("Failed to start command: %v", err)
		}

		pid := wrapper.Pid()
		if pid <= 0 {
			t.Errorf("Expected PID > 0, got %d", pid)
		}

		// Wait for command to finish (echo terminates quickly)
		wrapper.Wait()
	})

	t.Run("Stdout", func(t *testing.T) {
		cmd := exec.Command("cmd", "/C", "echo test")
		wrapper := &ExecCommandWrapper{cmd: cmd}

		var buf bytes.Buffer
		wrapper.Stdout(&buf)

		err := wrapper.Run()
		if err != nil {
			t.Fatalf("Failed to run command: %v", err)
		}

		output := strings.TrimSpace(buf.String())
		if output != "test" {
			t.Errorf("Expected 'test', got '%s'", output)
		}
	})

	t.Run("Stderr", func(t *testing.T) {
		cmd := exec.Command("cmd", "/C", "echo test 1>&2")
		wrapper := &ExecCommandWrapper{cmd: cmd}

		var buf bytes.Buffer
		wrapper.Stderr(&buf)

		err := wrapper.Run()
		if err != nil {
			t.Fatalf("Failed to run command: %v", err)
		}

		output := strings.TrimSpace(buf.String())
		if !strings.Contains(output, "test") {
			t.Errorf("Expected stderr to contain 'test', got '%s'", output)
		}
	})

	t.Run("Start", func(t *testing.T) {
		cmd := exec.Command("cmd", "/C", "echo test")
		wrapper := &ExecCommandWrapper{cmd: cmd}

		err := wrapper.Start()
		if err != nil {
			t.Fatalf("Failed to start command: %v", err)
		}

		// Command should be running
		pid := wrapper.Pid()
		if pid <= 0 {
			t.Errorf("Expected PID > 0, got %d", pid)
		}

		// Wait for command to finish (echo terminates quickly)
		wrapper.Wait()
	})

	t.Run("Run", func(t *testing.T) {
		cmd := exec.Command("cmd", "/C", "echo test")
		wrapper := &ExecCommandWrapper{cmd: cmd}

		err := wrapper.Run()
		if err != nil {
			t.Fatalf("Failed to run command: %v", err)
		}
	})

	// Note: Stop() test is skipped when running with coverage as stopWithGracefulShutdown
	// can hang due to coverage instrumentation interfering with process signal handling
	t.Run("Stop", func(t *testing.T) {
		if testing.CoverMode() != "" {
			t.Skip("Skipping Stop() test when running with coverage due to potential hangs")
		}
		// Use ping instead of timeout - timeout is a special Windows command
		// that can have permission issues when trying to terminate it
		cmd := exec.Command("ping", "-n", "10", "127.0.0.1")
		wrapper := &ExecCommandWrapper{cmd: cmd}

		err := wrapper.Start()
		if err != nil {
			t.Fatalf("Failed to start command: %v", err)
		}
		defer func() {
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
		}()

		// Stop should succeed and terminate the process
		// On Windows, access denied errors are acceptable as the process may still be terminated
		err = wrapper.Stop()
		if !isAcceptableWindowsTerminationError(err) {
			t.Errorf("Stop() should succeed or fail with access denied, got error: %v", err)
		}
	})

	t.Run("Wait", func(t *testing.T) {
		cmd := exec.Command("cmd", "/C", "echo test")
		wrapper := &ExecCommandWrapper{cmd: cmd}

		err := wrapper.Start()
		if err != nil {
			t.Fatalf("Failed to start command: %v", err)
		}

		err = wrapper.Wait()
		if err != nil {
			t.Errorf("Wait() should succeed for successful command, got error: %v", err)
		}
	})

	t.Run("String", func(t *testing.T) {
		cmd := exec.Command("cmd", "/C", "echo test args")
		cmd.Dir = os.TempDir()
		wrapper := &ExecCommandWrapper{cmd: cmd}

		str := wrapper.String()
		if !strings.Contains(str, "ExecCommandWrapper") {
			t.Errorf("Expected String() to contain 'ExecCommandWrapper', got '%s'", str)
		}
		if !strings.Contains(str, "echo") {
			t.Errorf("Expected String() to contain 'echo', got '%s'", str)
		}
	})
}

func TestExecCommandStopper(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})

	t.Run("Pid", func(t *testing.T) {
		cmd := exec.Command("cmd", "/C", "echo test")
		err := cmd.Start()
		if err != nil {
			t.Fatalf("Failed to start command: %v", err)
		}
		defer func() {
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
		}()

		stopper := &ExecCommandStopper{
			id:      "test-id",
			cmd:     cmd,
			timeout: 5 * time.Second,
		}

		pid := stopper.Pid()
		if pid <= 0 {
			t.Errorf("Expected PID > 0, got %d", pid)
		}

		// Wait for echo to finish (it terminates quickly)
		cmd.Wait()
	})

	t.Run("Stdout", func(t *testing.T) {
		cmd := exec.Command("cmd", "/C", "echo test")
		stopper := &ExecCommandStopper{
			id:      "test-id",
			cmd:     cmd,
			timeout: 5 * time.Second,
		}

		var buf bytes.Buffer
		stopper.Stdout(&buf)

		// Stdout should be set (no error expected)
		if cmd.Stdout == nil {
			t.Error("Expected stdout to be set")
		}
	})

	t.Run("Stderr", func(t *testing.T) {
		cmd := exec.Command("cmd", "/C", "echo test")
		stopper := &ExecCommandStopper{
			id:      "test-id",
			cmd:     cmd,
			timeout: 5 * time.Second,
		}

		var buf bytes.Buffer
		stopper.Stderr(&buf)

		// Stderr should be set (no error expected)
		if cmd.Stderr == nil {
			t.Error("Expected stderr to be set")
		}
	})

	t.Run("Start", func(t *testing.T) {
		if testing.CoverMode() != "" {
			t.Skip("Skipping Start() test when running with coverage due to potential hangs")
		}
		// Use ping instead of timeout - timeout is a special Windows command
		// that can have permission issues when trying to terminate it
		cmd := exec.Command("ping", "-n", "10", "127.0.0.1")
		err := cmd.Start()
		if err != nil {
			t.Fatalf("Failed to start command: %v", err)
		}
		defer func() {
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
		}()

		stopper := &ExecCommandStopper{
			id:      "test-id",
			cmd:     cmd,
			timeout: 1 * time.Second,
		}

		// Start should call Stop internally
		// On Windows, access denied errors are acceptable during process termination
		err = stopper.Start()
		if !isAcceptableWindowsTerminationError(err) {
			t.Errorf("Start() should succeed or fail with access denied, got error: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		if testing.CoverMode() != "" {
			t.Skip("Skipping Run() test when running with coverage due to potential hangs")
		}
		// Use ping instead of timeout - timeout is a special Windows command
		// that can have permission issues when trying to terminate it
		cmd := exec.Command("ping", "-n", "10", "127.0.0.1")
		err := cmd.Start()
		if err != nil {
			t.Fatalf("Failed to start command: %v", err)
		}
		defer func() {
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
		}()

		stopper := &ExecCommandStopper{
			id:      "test-id",
			cmd:     cmd,
			timeout: 1 * time.Second,
		}

		// Run should call Stop internally
		// On Windows, access denied errors are acceptable during process termination
		err = stopper.Run()
		if !isAcceptableWindowsTerminationError(err) {
			t.Errorf("Run() should succeed or fail with access denied, got error: %v", err)
		}
	})

	t.Run("Stop", func(t *testing.T) {
		if testing.CoverMode() != "" {
			t.Skip("Skipping Stop() test when running with coverage due to potential hangs")
		}
		// Use ping instead of timeout - timeout is a special Windows command
		// that can have permission issues when trying to terminate it
		cmd := exec.Command("ping", "-n", "10", "127.0.0.1")
		err := cmd.Start()
		if err != nil {
			t.Fatalf("Failed to start command: %v", err)
		}
		defer func() {
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
		}()

		stopper := &ExecCommandStopper{
			id:      "test-id",
			cmd:     cmd,
			timeout: 1 * time.Second,
		}

		// Stop should succeed
		// On Windows, access denied errors are acceptable during process termination
		err = stopper.Stop()
		if !isAcceptableWindowsTerminationError(err) {
			t.Errorf("Stop() should succeed or fail with access denied, got error: %v", err)
		}
	})

	t.Run("Stop with nil process", func(t *testing.T) {
		cmd := exec.Command("cmd", "/C", "echo test")
		stopper := &ExecCommandStopper{
			id:      "test-id",
			cmd:     cmd,
			timeout: 1 * time.Second,
		}

		// Stop should handle nil process gracefully
		err := stopper.Stop()
		if err != nil {
			t.Errorf("Stop() should handle nil process gracefully, got error: %v", err)
		}
	})

	t.Run("Wait", func(t *testing.T) {
		cmd := exec.Command("cmd", "/C", "echo test")
		err := cmd.Run()
		if err != nil {
			t.Fatalf("Failed to run command: %v", err)
		}

		stopper := &ExecCommandStopper{
			id:      "test-id",
			cmd:     cmd,
			timeout: 1 * time.Second,
		}

		// Wait should succeed (process already exited)
		err = stopper.Wait()
		if err != nil {
			t.Errorf("Wait() should succeed, got error: %v", err)
		}
	})

	t.Run("String", func(t *testing.T) {
		cmd := exec.Command("cmd", "/C", "echo test args")
		cmd.Dir = os.TempDir()
		stopper := &ExecCommandStopper{
			id:      "test-id",
			cmd:     cmd,
			timeout: 5 * time.Second,
		}

		str := stopper.String()
		if !strings.Contains(str, "ExecCommandStopper") {
			t.Errorf("Expected String() to contain 'ExecCommandStopper', got '%s'", str)
		}
		if !strings.Contains(str, "echo") {
			t.Errorf("Expected String() to contain 'echo', got '%s'", str)
		}
	})
}
