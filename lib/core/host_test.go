package core

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

type expectedHostProcess struct {
	id            string
	commandLine   string
	executable    string
	args          []string
	shell         Shell
	workingDir    string
	env           map[string]string
	vars          map[string]string
	awaitTimeout  string
	awaitResource string
}

func assertHostProcess(actual *HostProcess, expected expectedHostProcess, t *testing.T) {
	if actual.ID() != expected.id {
		t.Errorf(`Container id, expected %s, got %s`, expected.id, actual.ID())
	}
	assertSliceEquals(actual.Args, expected.args, `Args`, t)
	if actual.WorkingDir != expected.workingDir {
		t.Errorf(`Container working dir, expected %s, got %s`, expected.workingDir, actual.WorkingDir)
	}
	actualCommandLine := strings.TrimSpace(actual.CommandLine)
	if actualCommandLine != expected.commandLine {
		t.Errorf(`Container command line, expected %s, got %s`, expected.commandLine, actualCommandLine)
	}

	if expected.awaitTimeout != "" {
		if actual.Await.Timeout != expected.awaitTimeout {
			t.Errorf(`Container await timeout, expected %s, got %s`, expected.awaitTimeout, actual.Await.Timeout)
		}
	}
	if expected.awaitResource != "" {
		if actual.Await.Resource != expected.awaitResource {
			t.Errorf(`Container await resource, expected %s, got %s`, expected.awaitResource, actual.Await.Resource)
		}
	}

	// if sh, args, err := p.resolveShell()...
}

func TestHostProcess01(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})
	spec := `
command: echo 'Hello!'
shell:
  path: /bin/sh
  args:
    - "-c"
---`

	p := &HostProcess{}
	err := unmarshalStrict([]byte(spec), &p)

	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}

	expected := expectedHostProcess{
		commandLine: `echo 'Hello!'`,
	}

	assertHostProcess(p, expected, t)
}

func TestHostProcessEnvWithVars(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})

	spec := `
command: "echo VARS_IN_ENV $VARS_IN_ENV"
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
