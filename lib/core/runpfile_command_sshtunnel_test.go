package core

import (
	"bytes"
	"strings"
	"testing"
)

func TestSSHTunnelCommandWrapper(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})

	t.Run("Pid", func(t *testing.T) {
		wrapper := &SSHTunnelCommandWrapper{
			localAddress:  "localhost:8080",
			jumpAddress:   "jump:22",
			targetAddress: "target:80",
		}

		pid := wrapper.Pid()
		if pid != -100 {
			t.Errorf("Expected PID -100, got %d", pid)
		}
	})

	t.Run("Stdout", func(t *testing.T) {
		wrapper := &SSHTunnelCommandWrapper{
			localAddress:  "localhost:8080",
			jumpAddress:   "jump:22",
			targetAddress: "target:80",
		}

		var buf bytes.Buffer
		wrapper.Stdout(&buf)

		if wrapper.stdout == nil {
			t.Error("Expected stdout to be set")
		}
	})

	t.Run("Stderr", func(t *testing.T) {
		wrapper := &SSHTunnelCommandWrapper{
			localAddress:  "localhost:8080",
			jumpAddress:   "jump:22",
			targetAddress: "target:80",
		}

		var buf bytes.Buffer
		wrapper.Stderr(&buf)

		if wrapper.stderr == nil {
			t.Error("Expected stderr to be set")
		}
	})

	t.Run("Start", func(t *testing.T) {
		wrapper := &SSHTunnelCommandWrapper{
			localAddress:  "localhost:8080",
			jumpAddress:   "jump:22",
			targetAddress: "target:80",
		}

		var buf bytes.Buffer
		wrapper.Stdout(&buf)

		err := wrapper.Start()
		if err != nil {
			t.Errorf("Start() should succeed, got error: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		wrapper := &SSHTunnelCommandWrapper{
			localAddress:  "localhost:8080",
			jumpAddress:   "jump:22",
			targetAddress: "target:80",
		}

		err := wrapper.Run()
		if err != nil {
			t.Errorf("Run() should succeed, got error: %v", err)
		}
	})

	t.Run("Stop", func(t *testing.T) {
		wrapper := &SSHTunnelCommandWrapper{
			localAddress:  "localhost:8080",
			jumpAddress:   "jump:22",
			targetAddress: "target:80",
		}

		var buf bytes.Buffer
		wrapper.Stdout(&buf)

		// Stop should succeed even with nil connections
		err := wrapper.Stop()
		if err != nil {
			t.Errorf("Stop() should succeed, got error: %v", err)
		}
	})

	t.Run("String", func(t *testing.T) {
		wrapper := &SSHTunnelCommandWrapper{
			localAddress:  "localhost:8080",
			jumpAddress:   "jump:22",
			targetAddress: "target:80",
		}

		str := wrapper.String()
		if str == "" {
			t.Error("Expected String() to return non-empty string")
		}
		if !strings.Contains(str, "SSHTunnelCommandWrapper") {
			t.Errorf("Expected String() to contain 'SSHTunnelCommandWrapper', got '%s'", str)
		}
	})
}

func TestSSHTunnelCommandStopper(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})

	t.Run("Pid", func(t *testing.T) {
		wrapper := &SSHTunnelCommandWrapper{
			localAddress:  "localhost:8080",
			jumpAddress:   "jump:22",
			targetAddress: "target:80",
		}
		stopper := &SSHTunnelCommandStopper{
			id:  "test-id",
			cmd: wrapper,
		}

		pid := stopper.Pid()
		if pid != -300 {
			t.Errorf("Expected PID -300, got %d", pid)
		}
	})

	t.Run("Stdout", func(t *testing.T) {
		wrapper := &SSHTunnelCommandWrapper{
			localAddress:  "localhost:8080",
			jumpAddress:   "jump:22",
			targetAddress: "target:80",
		}
		stopper := &SSHTunnelCommandStopper{
			id:  "test-id",
			cmd: wrapper,
		}

		var buf bytes.Buffer
		stopper.Stdout(&buf)

		// Should not panic
		if stopper == nil {
			t.Error("Stopper should not be nil")
		}
	})

	t.Run("Stderr", func(t *testing.T) {
		wrapper := &SSHTunnelCommandWrapper{
			localAddress:  "localhost:8080",
			jumpAddress:   "jump:22",
			targetAddress: "target:80",
		}
		stopper := &SSHTunnelCommandStopper{
			id:  "test-id",
			cmd: wrapper,
		}

		var buf bytes.Buffer
		stopper.Stderr(&buf)

		// Should not panic
		if stopper == nil {
			t.Error("Stopper should not be nil")
		}
	})

	t.Run("Start", func(t *testing.T) {
		wrapper := &SSHTunnelCommandWrapper{
			localAddress:  "localhost:8080",
			jumpAddress:   "jump:22",
			targetAddress: "target:80",
		}
		stopper := &SSHTunnelCommandStopper{
			id:  "test-id",
			cmd: wrapper,
		}

		err := stopper.Start()
		if err != nil {
			t.Errorf("Start() should succeed, got error: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		wrapper := &SSHTunnelCommandWrapper{
			localAddress:  "localhost:8080",
			jumpAddress:   "jump:22",
			targetAddress: "target:80",
		}
		stopper := &SSHTunnelCommandStopper{
			id:  "test-id",
			cmd: wrapper,
		}

		err := stopper.Run()
		if err != nil {
			t.Errorf("Run() should succeed, got error: %v", err)
		}
	})

	t.Run("Stop", func(t *testing.T) {
		wrapper := &SSHTunnelCommandWrapper{
			localAddress:  "localhost:8080",
			jumpAddress:   "jump:22",
			targetAddress: "target:80",
		}
		stopper := &SSHTunnelCommandStopper{
			id:  "test-id",
			cmd: wrapper,
		}

		err := stopper.Stop()
		if err != nil {
			t.Errorf("Stop() should succeed, got error: %v", err)
		}
	})

	t.Run("Wait", func(t *testing.T) {
		wrapper := &SSHTunnelCommandWrapper{
			localAddress:  "localhost:8080",
			jumpAddress:   "jump:22",
			targetAddress: "target:80",
		}
		stopper := &SSHTunnelCommandStopper{
			id:  "test-id",
			cmd: wrapper,
		}

		err := stopper.Wait()
		if err != nil {
			t.Errorf("Wait() should succeed, got error: %v", err)
		}
	})

	t.Run("String", func(t *testing.T) {
		wrapper := &SSHTunnelCommandWrapper{
			localAddress:  "localhost:8080",
			jumpAddress:   "jump:22",
			targetAddress: "target:80",
		}
		stopper := &SSHTunnelCommandStopper{
			id:  "test-id",
			cmd: wrapper,
		}

		str := stopper.String()
		if str == "" {
			t.Error("Expected String() to return non-empty string")
		}
		if !strings.Contains(str, "SSHTunnelCommandStopper") {
			t.Errorf("Expected String() to contain 'SSHTunnelCommandStopper', got '%s'", str)
		}
	})
}
