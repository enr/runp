package core

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestExecCommandWrapper(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})

	t.Run("Pid", func(t *testing.T) {
		cmd := exec.Command("echo", "test")
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
		cmd := exec.Command("echo", "test")
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
		cmd := exec.Command("sh", "-c", "echo test >&2")
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
		cmd := exec.Command("echo", "test")
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
		cmd := exec.Command("echo", "test")
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
		cmd := exec.Command("sleep", "10")
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
		err = wrapper.Stop()
		if err != nil {
			t.Errorf("Stop() should succeed, got error: %v", err)
		}
	})

	t.Run("Wait", func(t *testing.T) {
		cmd := exec.Command("echo", "test")
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
		cmd := exec.Command("echo", "test", "args")
		cmd.Dir = "/tmp"
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
		cmd := exec.Command("echo", "test")
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
		cmd := exec.Command("echo", "test")
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
		cmd := exec.Command("echo", "test")
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
		cmd := exec.Command("sleep", "10")
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
		err = stopper.Start()
		if err != nil {
			t.Errorf("Start() should succeed, got error: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		if testing.CoverMode() != "" {
			t.Skip("Skipping Run() test when running with coverage due to potential hangs")
		}
		cmd := exec.Command("sleep", "10")
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
		err = stopper.Run()
		if err != nil {
			t.Errorf("Run() should succeed, got error: %v", err)
		}
	})

	t.Run("Stop", func(t *testing.T) {
		if testing.CoverMode() != "" {
			t.Skip("Skipping Stop() test when running with coverage due to potential hangs")
		}
		cmd := exec.Command("sleep", "10")
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
		err = stopper.Stop()
		if err != nil {
			t.Errorf("Stop() should succeed, got error: %v", err)
		}
	})

	t.Run("Stop with nil process", func(t *testing.T) {
		cmd := exec.Command("echo", "test")
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
		cmd := exec.Command("echo", "test")
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
		cmd := exec.Command("echo", "test", "args")
		cmd.Dir = "/tmp"
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
