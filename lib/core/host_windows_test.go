//go:build windows
// +build windows

package core

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestHostProcessEnvWithVarsWindows(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})

	spec := `
command: "echo VARS_IN_ENV %VARS_IN_ENV%"
env:
  VARS_IN_ENV: "{{vars runp_root}}"
---`

	p := &HostProcess{}
	err := unmarshalStrict([]byte(spec), &p)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	// Setup vars including runp_root
	runpRoot, err := filepath.Abs("/tmp/test-runp-root")
	if err != nil {
		t.Fatalf("Error creating test runp_root path: %v", err)
	}
	p.vars = map[string]string{
		"runp_root": runpRoot,
	}
	p.SetID("test-process")

	// Get the command
	cmdWrapper, err := p.StartCommand()
	if err != nil {
		t.Fatalf("Error creating command: %v", err)
	}

	// Capture stdout
	var stdout bytes.Buffer
	cmdWrapper.Stdout(&stdout)
	cmdWrapper.Stderr(&bytes.Buffer{})

	// Run the command
	err = cmdWrapper.Run()
	if err != nil {
		t.Fatalf("Error running command: %v", err)
	}

	// Verify output contains the substituted value
	output := stdout.String()
	expectedValue := runpRoot
	if !strings.Contains(output, expectedValue) {
		t.Errorf("Expected output to contain '%s', but got: %s", expectedValue, output)
	}

	// Verify the exact format
	expectedOutput := "VARS_IN_ENV " + runpRoot
	actualOutput := strings.TrimSpace(output)
	if actualOutput != expectedOutput {
		t.Errorf("Expected output '%s', got '%s'", expectedOutput, actualOutput)
	}
}
