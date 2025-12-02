package core

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"
)

// mockRunpCommand è un mock di RunpCommand per i test
type mockRunpCommand struct {
	pid      int
	startErr error
	waitErr  error
	stopErr  error
	started  bool
	waited   bool
	stopped  bool
	stdout   bytes.Buffer
	stderr   bytes.Buffer
}

func (m *mockRunpCommand) Pid() int {
	return m.pid
}

func (m *mockRunpCommand) Stdout(stdout io.Writer) {
	if stdout != nil {
		m.stdout.Reset()
		m.stdout.WriteString("mock stdout")
	}
}

func (m *mockRunpCommand) Stderr(stderr io.Writer) {
	if stderr != nil {
		m.stderr.Reset()
		m.stderr.WriteString("mock stderr")
	}
}

func (m *mockRunpCommand) Start() error {
	if m.startErr != nil {
		return m.startErr
	}
	m.started = true
	return nil
}

func (m *mockRunpCommand) Run() error {
	if err := m.Start(); err != nil {
		return err
	}
	return m.Wait()
}

func (m *mockRunpCommand) Stop() error {
	if m.stopErr != nil {
		return m.stopErr
	}
	m.stopped = true
	return nil
}

func (m *mockRunpCommand) Wait() error {
	if m.waitErr != nil {
		return m.waitErr
	}
	m.waited = true
	return nil
}

// mockRunpProcess è un mock di RunpProcess per i test
type mockRunpProcess struct {
	id            string
	dir           string
	shouldWait    bool
	awaitResource string
	awaitTimeout  string
	startable     bool
	startableErr  error
	startCmd      RunpCommand
	startCmdErr   error
	stopCmd       RunpCommand
	stopCmdErr    error
	preconditions Preconditions
}

func (m *mockRunpProcess) ID() string {
	return m.id
}

func (m *mockRunpProcess) SetID(id string) {
	m.id = id
}

func (m *mockRunpProcess) VerifyPreconditions() PreconditionVerifyResult {
	return m.preconditions.Verify()
}

func (m *mockRunpProcess) SetPreconditions(preconditions Preconditions) {
	m.preconditions = preconditions
}

func (m *mockRunpProcess) StartCommand() (RunpCommand, error) {
	if m.startCmdErr != nil {
		return nil, m.startCmdErr
	}
	return m.startCmd, nil
}

func (m *mockRunpProcess) StopCommand() (RunpCommand, error) {
	if m.stopCmdErr != nil {
		return nil, m.stopCmdErr
	}
	return m.stopCmd, nil
}

func (m *mockRunpProcess) StopTimeout() time.Duration {
	return 5 * time.Second
}

func (m *mockRunpProcess) Dir() string {
	return m.dir
}

func (m *mockRunpProcess) SetDir(dir string) {
	m.dir = dir
}

func (m *mockRunpProcess) ShouldWait() bool {
	return m.shouldWait
}

func (m *mockRunpProcess) AwaitResource() string {
	return m.awaitResource
}

func (m *mockRunpProcess) AwaitTimeout() string {
	return m.awaitTimeout
}

func (m *mockRunpProcess) IsStartable() (bool, error) {
	return m.startable, m.startableErr
}

func TestRunpfileExecutor_longestName(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: false,
		Color: false,
	})

	rf := &Runpfile{
		Units: map[string]*RunpUnit{
			"short":        {Name: "short"},
			"verylongname": {Name: "verylongname"},
			"medium":       {Name: "medium"},
		},
	}
	executor := NewExecutor(rf)
	longest := executor.longestName()
	if longest != 12 {
		t.Errorf("Expected longest name length 12, got %d", longest)
	}
}

func TestRunpfileExecutor_longestName_Cached(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: false,
		Color: false,
	})

	rf := &Runpfile{
		Units: map[string]*RunpUnit{
			"test": {Name: "test"},
		},
	}
	executor := NewExecutor(rf)
	executor.longest = 10
	longest := executor.longestName()
	if longest != 10 {
		t.Errorf("Expected cached longest name length 10, got %d", longest)
	}
}

func TestRunpfileExecutor_setupProcessCommand(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: false,
		Color: false,
	})

	rf := &Runpfile{}
	executor := NewExecutor(rf)
	appContext := GetApplicationContext()

	mockCmd := &mockRunpCommand{pid: 123}
	mockProcess := &mockRunpProcess{
		id:        "test-process",
		startCmd:  mockCmd,
		startable: true,
	}

	unit := &RunpUnit{
		Name: "test-unit",
	}

	logger := testLogger
	cmd, err := executor.setupProcessCommand(unit, mockProcess, logger, appContext)
	if err != nil {
		t.Errorf("setupProcessCommand() error = %v", err)
	}
	if cmd == nil {
		t.Error("setupProcessCommand() should return non-nil command")
	}
}

func TestRunpfileExecutor_setupProcessCommand_Error(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: false,
		Color: false,
	})

	rf := &Runpfile{}
	executor := NewExecutor(rf)
	appContext := GetApplicationContext()

	expectedErr := errors.New("command build error")
	mockProcess := &mockRunpProcess{
		id:          "test-process",
		startCmdErr: expectedErr,
		startable:   true,
	}

	unit := &RunpUnit{
		Name: "test-unit",
	}

	logger := testLogger
	cmd, err := executor.setupProcessCommand(unit, mockProcess, logger, appContext)
	if err == nil {
		t.Error("setupProcessCommand() should return error")
	}
	if cmd != nil {
		t.Error("setupProcessCommand() should return nil command on error")
	}
}

func TestRunpfileExecutor_verifyProcessStartability(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: false,
		Color: false,
	})

	rf := &Runpfile{}
	executor := NewExecutor(rf)
	appContext := GetApplicationContext()

	tests := []struct {
		name      string
		startable bool
		startErr  error
		wantErr   bool
	}{
		{
			name:      "startable",
			startable: true,
			wantErr:   false,
		},
		{
			name:      "not startable",
			startable: false,
			wantErr:   true,
		},
		{
			name:      "error checking startability",
			startable: false,
			startErr:  errors.New("check error"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProcess := &mockRunpProcess{
				id:           "test-process",
				startable:    tt.startable,
				startableErr: tt.startErr,
			}
			logger := testLogger
			err := executor.verifyProcessStartability(mockProcess, logger, appContext)
			if (err != nil) != tt.wantErr {
				t.Errorf("verifyProcessStartability() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunpfileExecutor_isGracefulShutdown(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: false,
		Color: false,
	})

	rf := &Runpfile{}
	executor := NewExecutor(rf)
	mockProcess := &mockRunpProcess{id: "test-process"}
	logger := testLogger

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "not an ExitError",
			err:      errors.New("generic error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.isGracefulShutdown(tt.err, mockProcess, logger)
			if result != tt.expected {
				t.Errorf("isGracefulShutdown() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRunpfileExecutor_isGracefulShutdown_ExitError(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: false,
		Color: false,
	})

	rf := &Runpfile{}
	executor := NewExecutor(rf)
	mockProcess := &mockRunpProcess{id: "test-process"}
	logger := testLogger

	// Test con un vero ExitError creato eseguendo un comando che fallisce
	// Nota: questo test potrebbe non funzionare su tutti i sistemi
	cmd := exec.Command("sh", "-c", "exit 143") // 143 = 128 + 15 (SIGTERM)
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Verifica che il codice di uscita sia quello atteso
			exitCode := exitErr.ExitCode()
			if exitCode == 143 {
				result := executor.isGracefulShutdown(exitErr, mockProcess, logger)
				if !result {
					t.Errorf("isGracefulShutdown() should return true for exit code 143 (SIGTERM), got %v", result)
				}
			} else {
				t.Logf("Exit code was %d, expected 143", exitCode)
			}
		} else {
			t.Logf("Error is not an ExitError: %T", err)
		}
	} else {
		t.Log("Command did not fail as expected")
	}
}

func TestRunpfileExecutor_handleProcessError(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: false,
		Color: false,
	})

	rf := &Runpfile{}
	executor := NewExecutor(rf)
	appContext := GetApplicationContext()
	mockProcess := &mockRunpProcess{id: "test-process"}
	logger := testLogger

	tests := []struct {
		name string
		err  error
	}{
		{
			name: "SyscallError",
			err:  &os.SyscallError{Syscall: "test", Err: errors.New("syscall error")},
		},
		{
			name: "generic error",
			err:  errors.New("generic error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor.handleProcessError(tt.err, mockProcess, logger, appContext)
			// Verifica che l'errore sia stato registrato
			reports := appContext.GetReport()
			if len(reports) == 0 {
				t.Error("handleProcessError() should add report")
			}
		})
	}
}

func TestRunpfileExecutor_readProcessOutput(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: false,
		Color: false,
	})

	rf := &Runpfile{}
	executor := NewExecutor(rf)
	mockProcess := &mockRunpProcess{id: "test-process"}
	logger := testLogger

	// Crea un pipe per simulare l'output del processo
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()

	// Scrivi alcuni dati nel pipe
	testData := "line 1\nline 2\nline 3\n"
	go func() {
		w.WriteString(testData)
		w.Close()
	}()

	// Aspetta un po' per permettere la scrittura
	time.Sleep(100 * time.Millisecond)

	executor.readProcessOutput(r, mockProcess, logger)

	// Verifica che l'output sia stato letto
	outputLines := testLogger.outputLines()
	if len(outputLines) == 0 {
		t.Error("readProcessOutput() should read and log output")
	}
}

func TestRunpfileExecutor_startProcessCommand(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: false,
		Color: false,
	})

	rf := &Runpfile{}
	executor := NewExecutor(rf)
	appContext := GetApplicationContext()
	mockProcess := &mockRunpProcess{id: "test-process"}
	logger := testLogger

	tests := []struct {
		name    string
		cmd     RunpCommand
		wantErr bool
	}{
		{
			name:    "success",
			cmd:     &mockRunpCommand{},
			wantErr: false,
		},
		{
			name:    "start error",
			cmd:     &mockRunpCommand{startErr: errors.New("start failed")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unit := &RunpUnit{Name: "test-unit"}
			r, w, _ := os.Pipe()
			defer r.Close()
			defer w.Close()

			var pwg sync.WaitGroup
			pwg.Add(1)

			err := executor.startProcessCommand(tt.cmd, unit, mockProcess, logger, appContext, w, &pwg)
			if (err != nil) != tt.wantErr {
				t.Errorf("startProcessCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunpfileExecutor_handleAwaitResources(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: false,
		Color: false,
	})

	rf := &Runpfile{}
	executor := NewExecutor(rf)
	appContext := GetApplicationContext()
	logger := testLogger

	tests := []struct {
		name         string
		process      RunpProcess
		wantErr      bool
		shouldWait   bool
		awaitTimeout string
	}{
		{
			name:         "no await",
			process:      &mockRunpProcess{shouldWait: false},
			wantErr:      false,
			shouldWait:   false,
			awaitTimeout: "",
		},
		{
			name:         "await with valid timeout",
			process:      &mockRunpProcess{shouldWait: true, awaitTimeout: "100ms"},
			wantErr:      false,
			shouldWait:   true,
			awaitTimeout: "100ms",
		},
		{
			name:         "await with invalid timeout",
			process:      &mockRunpProcess{shouldWait: true, awaitTimeout: "invalid"},
			wantErr:      true,
			shouldWait:   true,
			awaitTimeout: "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset appContext per ogni test
			appContext = GetApplicationContext()
			err := executor.handleAwaitResources(tt.process, logger, appContext)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleAwaitResources() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
