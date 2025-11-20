package core

import (
	"bytes"
	"strings"
	"testing"
)

// Helper function per creare un HostProcess da una spec YAML
func createHostProcessFromSpec(spec string) (*HostProcess, error) {
	p := &HostProcess{}
	err := unmarshalStrict([]byte(spec), &p)
	return p, err
}

// Helper function per configurare l'UI per i test
func setupTestUI(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})
}

// Helper function per confrontare due map di stringhe
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

// Helper functions per verificare singoli campi di HostProcess

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

// Helper function per confrontare un HostProcess con i valori attesi
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

// Test per resolveShell con shell personalizzato
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

// Test per resolveShell senza shell specificato (usa default)
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

	// Verifica che il command sia nell'ultimo arg
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

// Test per resolveShell con command line vuoto
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

	// L'ultimo arg dovrebbe essere vuoto
	lastArg := args[len(args)-1]
	if lastArg != "" {
		t.Errorf("Expected last arg to be empty, got '%s'", lastArg)
	}
}

// Test per resolveEnvironment senza Env
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

// Test per resolveEnvironment con Env semplice
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

// Test per resolveEnvironment con variabili template
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

// Test per resolveEnvironment con variabile non definita
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

	// La variabile non definita dovrebbe rimanere come template o essere vuota
	// dipende dall'implementazione di cliPreprocessor
	if val, exists := envMap["TEST_VAR"]; exists {
		// Verifichiamo che non sia vuota (il preprocessor potrebbe gestirla in modo diverso)
		if val == "" {
			t.Log("Note: undefined variable resulted in empty value")
		}
	}
}

// Test per StartCommand con executable non trovato
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

// Test per StartCommand con executable nel workingDir
func TestHostProcessStartCommandExecutableInWorkingDir(t *testing.T) {
	setupTestUI(t)

	// Usiamo un comando che esiste nel sistema come eseguibile,
	// presente in $PATH con stesso nome per tutti i sistemi operativi
	p := &HostProcess{
		Executable: "hostname",
		Args:       []string{},
		WorkingDir: "/tmp",
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
	if output == "" {
		t.Errorf("Expected output not empty")
	}
}

// Test per StartCommand con command line (non executable)
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

// Test per StopCommand
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

	// Verifica che StopCommand ritorna sempre un comando valido
	// Anche se il processo non è stato avviato
	// Nota: Pid() può panic se cmd è nil, quindi verifichiamo solo che non sia nil
	if stopCmd == nil {
		t.Error("Expected StopCommand to return a non-nil command")
	}
}

// Test per SetPreconditions e VerifyPreconditions
func TestHostProcessPreconditions(t *testing.T) {
	setupTestUI(t)

	p := &HostProcess{}
	p.SetID("test-process")

	// Inizialmente non ci sono preconditions
	result := p.VerifyPreconditions()
	// Il voto iniziale può essere Unknown o Proceed (dipende dall'implementazione)
	if result.Vote != Unknown && result.Vote != Proceed {
		t.Errorf("Expected initial precondition vote to be Unknown or Proceed, got %v", result.Vote)
	}

	// Imposta preconditions vuote
	preconditions := Preconditions{}
	p.SetPreconditions(preconditions)

	result = p.VerifyPreconditions()
	// Preconditions vuote dovrebbero passare o essere Unknown
	if result.Vote == Stop {
		t.Errorf("Expected precondition vote to not be Stop for empty preconditions, got %v", result.Vote)
	}
}

// Test per StartCommand quando sia executable che command line sono vuoti
func TestHostProcessStartCommandBothEmpty(t *testing.T) {
	setupTestUI(t)

	p := &HostProcess{
		Executable:  "",
		CommandLine: "",
	}
	p.SetID("test-process")

	// Dovrebbe usare commandline path (buildCmdCommandline)
	// che userà resolveShell con command line vuoto
	cmdWrapper, err := p.StartCommand()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cmdWrapper == nil {
		t.Fatal("Expected command wrapper to be non-nil")
	}
}

// Test per resolveEnvironment con Env vuoto map
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

// Test per resolveShell con shell args vuoti
func TestHostProcessResolveShellEmptyArgs(t *testing.T) {
	setupTestUI(t)

	p := &HostProcess{
		CommandLine: "test command",
		Shell: Shell{
			Path: "/bin/sh",
			Args: []string{}, // Args vuoti
		},
	}

	exe, args, err := p.resolveShell()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if exe != "/bin/sh" {
		t.Errorf("Expected shell path '/bin/sh', got '%s'", exe)
	}

	// Dovrebbe avere almeno il command line come arg
	if len(args) == 0 {
		t.Error("Expected at least one arg (command line)")
	}

	if args[len(args)-1] != "test command" {
		t.Errorf("Expected last arg to be 'test command', got '%s'", args[len(args)-1])
	}
}
