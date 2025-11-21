package core

import (
	"testing"
	"time"
)

// stubProcess is a simple stub implementation of RunpProcess for testing purposes
type stubProcess struct {
	id string
}

func (s *stubProcess) ID() string { return s.id }
func (s *stubProcess) VerifyPreconditions() PreconditionVerifyResult {
	return PreconditionVerifyResult{}
}
func (s *stubProcess) SetPreconditions(Preconditions)     {}
func (s *stubProcess) SetID(id string)                    { s.id = id }
func (s *stubProcess) StartCommand() (RunpCommand, error) { return nil, nil }
func (s *stubProcess) StopCommand() (RunpCommand, error)  { return nil, nil }
func (s *stubProcess) StopTimeout() time.Duration         { return 0 }
func (s *stubProcess) Dir() string                        { return "" }
func (s *stubProcess) SetDir(string)                      {}
func (s *stubProcess) ShouldWait() bool                   { return false }
func (s *stubProcess) AwaitResource() string              { return "" }
func (s *stubProcess) AwaitTimeout() string               { return "" }
func (s *stubProcess) IsStartable() (bool, error)         { return true, nil }

func TestGetRunningProcesses(t *testing.T) {
	ctx := GetApplicationContext()

	// Clear previous state
	ctx.runningProcesses = make(map[string]RunpProcess)

	// Verify initial state is empty
	processes := ctx.GetRunningProcesses()
	if processes == nil {
		t.Error("GetRunningProcesses should return a non-nil map")
	}
	if len(processes) != 0 {
		t.Errorf("Expected empty map, got %d processes", len(processes))
	}

	// Register a process
	proc1 := &stubProcess{id: "proc1"}
	ctx.RegisterRunningProcess(proc1)

	// Verify process is registered
	processes = ctx.GetRunningProcesses()
	if len(processes) != 1 {
		t.Errorf("Expected 1 process, got %d", len(processes))
	}
	if processes["proc1"] == nil {
		t.Error("Expected proc1 to be in the map")
	}

	// Register another process
	proc2 := &stubProcess{id: "proc2"}
	ctx.RegisterRunningProcess(proc2)

	processes = ctx.GetRunningProcesses()
	if len(processes) != 2 {
		t.Errorf("Expected 2 processes, got %d", len(processes))
	}
}

func TestGetReport(t *testing.T) {
	ctx := GetApplicationContext()

	// Clear previous state
	ctx.report = []string{}

	// Verify initial state is empty
	report := ctx.GetReport()
	if report == nil {
		t.Error("GetReport should return a non-nil slice")
	}
	if len(report) != 0 {
		t.Errorf("Expected empty report, got %d items", len(report))
	}

	// Add a report entry
	ctx.AddReport("message1")

	// Verify report entry is present
	report = ctx.GetReport()
	if len(report) != 1 {
		t.Errorf("Expected 1 report, got %d", len(report))
	}
	if report[0] != "message1" {
		t.Errorf("Expected 'message1', got '%s'", report[0])
	}
}

func TestAddReport(t *testing.T) {
	ctx := GetApplicationContext()

	// Clear previous state
	ctx.report = []string{}

	// Add multiple report entries
	ctx.AddReport("first message")
	ctx.AddReport("second message")
	ctx.AddReport("third message")

	// Verifica che tutti siano presenti
	report := ctx.GetReport()
	if len(report) != 3 {
		t.Errorf("Expected 3 reports, got %d", len(report))
	}
	if report[0] != "first message" {
		t.Errorf("Expected 'first message', got '%s'", report[0])
	}
	if report[1] != "second message" {
		t.Errorf("Expected 'second message', got '%s'", report[1])
	}
	if report[2] != "third message" {
		t.Errorf("Expected 'third message', got '%s'", report[2])
	}

	// Test con stringa vuota
	ctx.AddReport("")
	report = ctx.GetReport()
	if len(report) != 4 {
		t.Errorf("Expected 4 reports after adding empty string, got %d", len(report))
	}
	if report[3] != "" {
		t.Errorf("Expected empty string, got '%s'", report[3])
	}
}
