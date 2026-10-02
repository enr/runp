//go:build darwin || freebsd || linux || netbsd || openbsd
// +build darwin freebsd linux netbsd openbsd

package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newLifetimeExecutor(t *testing.T, spec string) *RunpfileExecutor {
	t.Helper()
	ConfigureUI(testLogger, LoggerConfig{Debug: false, Color: false})
	rf := &Runpfile{}
	if err := unmarshalStrict([]byte(spec), rf); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for name, unit := range rf.Units {
		unit.Name = name
		unit.Process().SetID(name)
	}
	return &RunpfileExecutor{
		rf:            rf,
		LoggerFactory: createStubLogger,
		newPipe:       os.Pipe,
	}
}

func startWithTimeout(t *testing.T, sut *RunpfileExecutor, timeout time.Duration) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- sut.Start() }()
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		t.Fatal("Start did not return in time")
		return nil
	}
}

// A unit with a ready condition must keep Start blocked until it exits,
// otherwise runp returns and leaves the process orphaned.
func TestStart_WaitsForReadyUnitsToExit(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "finished")
	sut := newLifetimeExecutor(t, `
units:
  svc:
    ready:
      output: "up"
      timeout: 5s
    host:
      command: "echo up && sleep 1 && touch `+marker+`"
`)
	if err := startWithTimeout(t, sut, 10*time.Second); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Errorf("Start returned before the ready unit exited: %v", err)
	}
}

// When a unit fails to start, the units already running must be stopped
// before Start returns.
func TestStart_StopsRunningUnitsWhenAUnitFails(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "survived")
	sut := newLifetimeExecutor(t, `
units:
  svc:
    ready:
      delay: 100ms
      timeout: 5s
    host:
      command: "sleep 5 && touch `+marker+`"
  broken:
    depends_on: [svc]
    host:
      executable: /nonexistent/executable/for/runp/test
`)
	start := time.Now()
	err := startWithTimeout(t, sut, 4*time.Second)
	if err == nil {
		t.Fatal("expected an error from Start")
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("running unit was not stopped: Start took %v", elapsed)
	}
	time.Sleep(100 * time.Millisecond)
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Error("running unit was not stopped")
	}
}

func TestGetRunningProcesses_ReturnsCopy(t *testing.T) {
	ctx := &ApplicationContext{runningProcesses: map[string]RunpProcess{}}
	ctx.RegisterRunningProcess(&stubProcess{id: "a"})
	snapshot := ctx.GetRunningProcesses()
	ctx.RemoveRunningProcess(&stubProcess{id: "a"})
	if len(snapshot) != 1 {
		t.Errorf("snapshot changed after removal: %v", snapshot)
	}
}

func TestStopProcess_DoesNotSignalCallerGroup(t *testing.T) {
	// The child shares the test's process group (no Setpgid): stopping it
	// must not signal the group, which would terminate the test binary.
	w := &ExecCommandWrapper{cmd: exec.Command("sleep", "5")}
	if err := w.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- w.Wait() }()
	if err := w.stopWithGracefulShutdown(2*time.Second, "test"); err != nil {
		t.Errorf("stop: %v", err)
	}
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "terminated") {
			t.Errorf("expected process terminated by signal, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("process not stopped")
	}
}
