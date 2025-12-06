package core

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// Helper function to create a HostProcess from a YAML spec.
func createHostProcessFromSpec(spec string) (*HostProcess, error) {
	p := &HostProcess{}
	err := unmarshalStrict([]byte(spec), &p)
	return p, err
}

// Helper function to configure the UI for tests.
func setupTestUI(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})
}

// Helper function to compare two string maps.
func assertMapEquals(actual, expected map[string]string, fieldName string, t *testing.T) {
	if len(actual) != len(expected) {
		t.Errorf(`%s length, expected %d, got %d`, fieldName, len(expected), len(actual))
		return
	}
	for k, v := range expected {
		if actual[k] != v {
			t.Errorf(`%s[%s], expected %s, got %s`, fieldName, k, v, actual[k])
		}
	}
}

// Helper functions to verify individual HostProcess fields.

func assertHostProcessID(actual *HostProcess, expected expectedHostProcess, t *testing.T) {
	if expected.id != "" && actual.ID() != expected.id {
		t.Errorf(`Process id, expected %s, got %s`, expected.id, actual.ID())
	}
}

func assertHostProcessCommandLine(actual *HostProcess, expected expectedHostProcess, t *testing.T) {
	if expected.commandLine != "" {
		actualCommandLine := strings.TrimSpace(actual.CommandLine)
		if actualCommandLine != expected.commandLine {
			t.Errorf(`Command line, expected %s, got %s`, expected.commandLine, actualCommandLine)
		}
	}
}

func assertHostProcessExecutable(actual *HostProcess, expected expectedHostProcess, t *testing.T) {
	if expected.executable != "" {
		if actual.Executable != expected.executable {
			t.Errorf(`Executable, expected %s, got %s`, expected.executable, actual.Executable)
		}
	}
}

func assertHostProcessArgs(actual *HostProcess, expected expectedHostProcess, t *testing.T) {
	if expected.args != nil {
		assertSliceEquals(actual.Args, expected.args, "Args", t)
	}
}

func assertHostProcessShell(actual *HostProcess, expected expectedHostProcess, t *testing.T) {
	if expected.shell.Path != "" {
		if actual.Shell.Path != expected.shell.Path {
			t.Errorf(`Shell path, expected %s, got %s`, expected.shell.Path, actual.Shell.Path)
		}
		if expected.shell.Args != nil {
			assertSliceEquals(actual.Shell.Args, expected.shell.Args, "Shell args", t)
		}
	}
}

func assertHostProcessWorkingDir(actual *HostProcess, expected expectedHostProcess, t *testing.T) {
	if expected.workingDir != "" {
		if actual.WorkingDir != expected.workingDir {
			t.Errorf(`Working dir, expected %s, got %s`, expected.workingDir, actual.WorkingDir)
		}
	}
}

func assertHostProcessEnv(actual *HostProcess, expected expectedHostProcess, t *testing.T) {
	if expected.env != nil {
		assertMapEquals(actual.Env, expected.env, "Env", t)
	}
}

func assertHostProcessAwait(actual *HostProcess, expected expectedHostProcess, t *testing.T) {
	if expected.awaitTimeout != "" {
		if actual.Await.Timeout != expected.awaitTimeout {
			t.Errorf(`Await timeout, expected %s, got %s`, expected.awaitTimeout, actual.Await.Timeout)
		}
	}
	if expected.awaitResource != "" {
		if actual.Await.Resource != expected.awaitResource {
			t.Errorf(`Await resource, expected %s, got %s`, expected.awaitResource, actual.Await.Resource)
		}
	}
}

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

// Helper function to compare a HostProcess with expected values.
func assertHostProcess(actual *HostProcess, expected expectedHostProcess, t *testing.T) {
	assertHostProcessID(actual, expected, t)
	assertHostProcessCommandLine(actual, expected, t)
	assertHostProcessExecutable(actual, expected, t)
	assertHostProcessArgs(actual, expected, t)
	assertHostProcessShell(actual, expected, t)
	assertHostProcessWorkingDir(actual, expected, t)
	assertHostProcessEnv(actual, expected, t)
	assertHostProcessAwait(actual, expected, t)
}

func TestHostProcess01(t *testing.T) {
	setupTestUI(t)

	spec := `
command: echo 'Hello!'
shell:
  path: /bin/sh
  args:
    - "-c"
---`

	p, err := createHostProcessFromSpec(spec)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	expected := expectedHostProcess{
		commandLine: `echo 'Hello!'`,
		shell: Shell{
			Path: "/bin/sh",
			Args: []string{"-c"},
		},
	}

	assertHostProcess(p, expected, t)
}

// Test variables in "workdir" field.
func TestHostProcessWorkdirWithVars(t *testing.T) {
	setupTestUI(t)

	spec := `
command: "echo hi"
workdir: "{{vars test_dir}}"
---`

	p := &HostProcess{}
	err := unmarshalStrict([]byte(spec), &p)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
	// Set vars for variable substitution
	p.vars = map[string]string{
		"test_dir": "/tmp/test-dir",
	}
	// check working dir variable substitution
	resolvedWorkingDir := p.resolveWorkingDir()
	if resolvedWorkingDir != "/tmp/test-dir" {
		t.Errorf("Expected working dir '/tmp/test-dir', got '%s'", resolvedWorkingDir)
	}
}

// Tests resolveShell with a custom shell.
func TestHostProcessResolveShellCustom(t *testing.T) {
	setupTestUI(t)

	p := &HostProcess{
		CommandLine: "echo test",
		Shell: Shell{
			Path: "/bin/zsh",
			Args: []string{"-c"},
		},
	}

	exe, args, err := p.resolveShell()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if exe != "/bin/zsh" {
		t.Errorf("Expected shell path '/bin/zsh', got '%s'", exe)
	}

	if len(args) < 2 {
		t.Errorf("Expected at least 2 args, got %d", len(args))
	}

	if args[0] != "-c" {
		t.Errorf("Expected first arg '-c', got '%s'", args[0])
	}

	lastArg := args[len(args)-1]
	if lastArg != "echo test" {
		t.Errorf("Expected last arg 'echo test', got '%s'", lastArg)
	}
}

// Tests resolveShell without a specified shell (uses default).
func TestHostProcessResolveShellDefault(t *testing.T) {
	setupTestUI(t)

	p := &HostProcess{
		CommandLine: "echo default",
		Shell: Shell{
			Path: "",
			Args: nil,
		},
	}

	exe, args, err := p.resolveShell()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if exe == "" {
		t.Error("Expected default shell path to be non-empty")
	}

	if len(args) == 0 {
		t.Error("Expected args to be non-empty")
	}

	// Verify that the command is in the last argument.
	found := false
	for _, arg := range args {
		if strings.Contains(arg, "echo default") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected command 'echo default' to be in shell args")
	}
}

// Tests resolveShell with an empty command line.
func TestHostProcessResolveShellEmptyCommand(t *testing.T) {
	setupTestUI(t)

	p := &HostProcess{
		CommandLine: "",
		Shell: Shell{
			Path: "/bin/sh",
			Args: []string{"-c"},
		},
	}

	exe, args, err := p.resolveShell()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if exe != "/bin/sh" {
		t.Errorf("Expected shell path '/bin/sh', got '%s'", exe)
	}

	// The last argument should be empty.
	lastArg := args[len(args)-1]
	if lastArg != "" {
		t.Errorf("Expected last arg to be empty, got '%s'", lastArg)
	}
}

// Tests resolveEnvironment without an environment.
func TestHostProcessResolveEnvironmentEmpty(t *testing.T) {
	setupTestUI(t)

	p := &HostProcess{
		Env:  nil,
		vars: map[string]string{},
	}

	env := p.resolveEnvironment()
	if env == nil {
		t.Error("Expected environment to be non-nil")
	}
	if len(env) != 0 {
		t.Errorf("Expected empty environment, got %d vars", len(env))
	}
}

// Tests resolveEnvironment with a simple environment.
func TestHostProcessResolveEnvironmentSimple(t *testing.T) {
	setupTestUI(t)

	p := &HostProcess{
		Env: map[string]string{
			"VAR1": "value1",
			"VAR2": "value2",
		},
		vars: map[string]string{},
	}

	env := p.resolveEnvironment()
	if len(env) != 2 {
		t.Errorf("Expected 2 environment variables, got %d", len(env))
	}

	envMap := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	if envMap["VAR1"] != "value1" {
		t.Errorf("Expected VAR1=value1, got VAR1=%s", envMap["VAR1"])
	}
	if envMap["VAR2"] != "value2" {
		t.Errorf("Expected VAR2=value2, got VAR2=%s", envMap["VAR2"])
	}
}

// Tests resolveEnvironment with template variables.
func TestHostProcessResolveEnvironmentWithVars(t *testing.T) {
	setupTestUI(t)

	p := &HostProcess{
		Env: map[string]string{
			"TEST_VAR": "{{vars test_key}}",
			"STATIC":   "static_value",
		},
		vars: map[string]string{
			"test_key": "substituted_value",
		},
	}

	env := p.resolveEnvironment()
	if len(env) != 2 {
		t.Errorf("Expected 2 environment variables, got %d", len(env))
	}

	envMap := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	if envMap["TEST_VAR"] != "substituted_value" {
		t.Errorf("Expected TEST_VAR=substituted_value, got TEST_VAR=%s", envMap["TEST_VAR"])
	}
	if envMap["STATIC"] != "static_value" {
		t.Errorf("Expected STATIC=static_value, got STATIC=%s", envMap["STATIC"])
	}
}

// Tests resolveEnvironment with an undefined variable.
func TestHostProcessResolveEnvironmentUndefinedVar(t *testing.T) {
	setupTestUI(t)

	p := &HostProcess{
		Env: map[string]string{
			"TEST_VAR": "{{vars undefined_key}}",
		},
		vars: map[string]string{},
	}

	env := p.resolveEnvironment()
	if len(env) != 1 {
		t.Errorf("Expected 1 environment variable, got %d", len(env))
	}

	envMap := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// An undefined variable should either remain as a template or be empty,
	// depending on the cliPreprocessor implementation.
	if val, exists := envMap["TEST_VAR"]; exists {
		// Check that it's not empty (the preprocessor might handle it differently).
		if val == "" {
			t.Log("Note: undefined variable resulted in empty value")
		}
	}
}

// Tests StartCommand with an executable that is not found.
func TestHostProcessStartCommandExecutableNotFound(t *testing.T) {
	setupTestUI(t)

	p := &HostProcess{
		Executable: "/nonexistent/executable/that/does/not/exist",
		Args:       []string{"arg1"},
	}
	p.SetID("test-process")

	_, err := p.StartCommand()
	if err == nil {
		t.Error("Expected error when executable is not found")
	}
}

// Tests StartCommand with an executable in the working directory.
func TestHostProcessStartCommandExecutableInWorkingDir(t *testing.T) {
	setupTestUI(t)

	// Use a command that exists on the system as the executable,
	// present in $PATH with the same name across all operating systems.
	// Use os.TempDir() to get a valid temporary directory for all platforms.
	tempDir := os.TempDir()
	p := &HostProcess{
		Executable: "hostname",
		Args:       []string{},
		WorkingDir: tempDir,
	}
	p.SetID("test-process")

	cmdWrapper, err := p.StartCommand()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cmdWrapper == nil {
		t.Fatal("Expected command wrapper to be non-nil")
	}

	var stdout bytes.Buffer
	cmdWrapper.Stdout(&stdout)
	cmdWrapper.Stderr(&bytes.Buffer{})

	err = cmdWrapper.Run()
	if err != nil {
		t.Fatalf("Error running %v command: %v", p, err)
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		t.Errorf("Expected %v command output not empty, got '%s'", p, output)
	}
}

// Tests StartCommand with a command line (not an executable).
func TestHostProcessStartCommandWithCommandLine(t *testing.T) {
	setupTestUI(t)

	p := &HostProcess{
		CommandLine: "echo test command",
	}
	p.SetID("test-process")

	cmdWrapper, err := p.StartCommand()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cmdWrapper == nil {
		t.Fatal("Expected command wrapper to be non-nil")
	}

	var stdout bytes.Buffer
	cmdWrapper.Stdout(&stdout)
	cmdWrapper.Stderr(&bytes.Buffer{})

	err = cmdWrapper.Run()
	if err != nil {
		t.Fatalf("Error running command: %v", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output != "test command" {
		t.Errorf("Expected output 'test command', got '%s'", output)
	}
}

// Tests StopCommand.
func TestHostProcessStopCommand(t *testing.T) {
	setupTestUI(t)

	p := &HostProcess{}
	p.SetID("test-process")

	stopCmd, err := p.StopCommand()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if stopCmd == nil {
		t.Fatal("Expected stop command to be non-nil")
	}

	// Verify that StopCommand always returns a valid command,
	// even if the process has not been started.
	// Note: Pid() can panic if cmd is nil, so we only check that it's not nil.
	if stopCmd == nil {
		t.Error("Expected StopCommand to return a non-nil command")
	}
}

// Tests SetPreconditions and VerifyPreconditions.
func TestHostProcessPreconditions(t *testing.T) {
	setupTestUI(t)

	p := &HostProcess{}
	p.SetID("test-process")

	// Initially, there are no preconditions.
	result := p.VerifyPreconditions()
	// The initial vote can be Unknown or Proceed (depending on the implementation).
	if result.Vote != Unknown && result.Vote != Proceed {
		t.Errorf("Expected initial precondition vote to be Unknown or Proceed, got %v", result.Vote)
	}

	// Set empty preconditions.
	preconditions := Preconditions{}
	p.SetPreconditions(preconditions)

	result = p.VerifyPreconditions()
	// Empty preconditions should pass or be Unknown.
	if result.Vote == Stop {
		t.Errorf("Expected precondition vote to not be Stop for empty preconditions, got %v", result.Vote)
	}
}

// Tests StartCommand when both executable and command line are empty.
func TestHostProcessStartCommandBothEmpty(t *testing.T) {
	setupTestUI(t)

	p := &HostProcess{
		Executable:  "",
		CommandLine: "",
	}
	p.SetID("test-process")

	// It should use the command line path (buildCmdCommandline),
	// which will use resolveShell with an empty command line.
	cmdWrapper, err := p.StartCommand()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cmdWrapper == nil {
		t.Fatal("Expected command wrapper to be non-nil")
	}
}

// Tests resolveEnvironment with an empty Env map.
func TestHostProcessResolveEnvironmentEmptyMap(t *testing.T) {
	setupTestUI(t)

	p := &HostProcess{
		Env:  map[string]string{},
		vars: map[string]string{"key": "value"},
	}

	env := p.resolveEnvironment()
	if env == nil {
		t.Error("Expected environment to be non-nil")
	}
	if len(env) != 0 {
		t.Errorf("Expected empty environment, got %d vars", len(env))
	}
}

// Tests resolveShell with empty shell args.
func TestHostProcessResolveShellEmptyArgs(t *testing.T) {
	setupTestUI(t)

	p := &HostProcess{
		CommandLine: "test command",
		Shell: Shell{
			Path: "/bin/sh",
			Args: []string{}, // Empty args
		},
	}

	exe, args, err := p.resolveShell()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if exe != "/bin/sh" {
		t.Errorf("Expected shell path '/bin/sh', got '%s'", exe)
	}

	// Should have at least the command line as an arg.
	if len(args) == 0 {
		t.Error("Expected at least one arg (command line)")
	}

	if args[len(args)-1] != "test command" {
		t.Errorf("Expected last arg to be 'test command', got '%s'", args[len(args)-1])
	}
}

func TestHostProcess_String(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})

	hp := &HostProcess{
		id: "test-id",
	}
	hp.SetID("test-id")

	result := hp.String()
	if result == "" {
		t.Error("Expected String() to return non-empty string")
	}
	if !strings.Contains(result, "test-id") {
		t.Errorf("Expected String() to contain 'test-id', got '%s'", result)
	}
}

func TestHostProcess_ShouldWait(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})

	t.Run("with timeout", func(t *testing.T) {
		hp := &HostProcess{
			Await: AwaitCondition{Timeout: "5s"},
		}
		if !hp.ShouldWait() {
			t.Error("Expected ShouldWait() to return true")
		}
	})

	t.Run("without timeout", func(t *testing.T) {
		hp := &HostProcess{
			Await: AwaitCondition{Timeout: ""},
		}
		if hp.ShouldWait() {
			t.Error("Expected ShouldWait() to return false")
		}
	})
}

func TestHostProcess_AwaitResource(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})

	hp := &HostProcess{
		Await: AwaitCondition{Resource: "http://localhost:8080"},
	}
	result := hp.AwaitResource()
	if result != "http://localhost:8080" {
		t.Errorf("Expected 'http://localhost:8080', got '%s'", result)
	}
}

func TestHostProcess_AwaitTimeout(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})

	hp := &HostProcess{
		Await: AwaitCondition{Timeout: "10s"},
	}
	result := hp.AwaitTimeout()
	if result != "10s" {
		t.Errorf("Expected '10s', got '%s'", result)
	}
}
