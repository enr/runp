//go:build darwin || freebsd || linux || netbsd || openbsd
// +build darwin freebsd linux netbsd openbsd

package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A dependency whose ready condition fails must not release its dependents.
func TestStart_FailedReadinessBlocksDependents(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "dependent-ran")
	sut := newLifetimeExecutor(t, `
units:
  svc:
    ready:
      output: "never printed"
      timeout: 5s
    host:
      command: "echo starting"
  dependent:
    depends_on: [svc]
    host:
      command: "touch `+marker+`"
`)
	err := startWithTimeout(t, sut, 10*time.Second)
	if err == nil {
		t.Error("expected an error when a dependency is never ready")
	}
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Error("dependent started although its dependency was never ready")
	}
}

// A dependency without ready condition that exits with an error must not
// release its dependents.
func TestStart_FailedDependencyExitBlocksDependents(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "dependent-ran")
	sut := newLifetimeExecutor(t, `
units:
  migrate:
    host:
      command: "exit 3"
  api:
    depends_on: [migrate]
    host:
      command: "touch `+marker+`"
`)
	err := startWithTimeout(t, sut, 10*time.Second)
	if err == nil || !strings.Contains(err.Error(), "failed") {
		t.Errorf("expected a failure, got %v", err)
	}
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Error("dependent started although its dependency failed")
	}
}

// A unit must start as soon as its own dependencies are ready, without
// waiting for unrelated units in the same "layer" to exit.
func TestStart_DependentsDoNotWaitForUnrelatedUnits(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "dependent-ran")
	sut := newLifetimeExecutor(t, `
units:
  svc:
    ready:
      delay: 100ms
    host:
      command: "sleep 3"
  unrelated:
    host:
      command: "sleep 3"
  dependent:
    depends_on: [svc]
    host:
      command: "touch `+marker+`"
`)
	done := make(chan error, 1)
	go func() { done <- sut.Start() }()

	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("dependent did not start while an unrelated unit was still running")
		}
		time.Sleep(20 * time.Millisecond)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Start did not return")
	}
}

// Vars in workdir are resolved when units start, with the final vars
// (runp_root and --var overrides are only set after loading).
func TestWorkdirVarsResolvedAtStart(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{Debug: false, Color: false})
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "other"), 0755); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(root, "pwd.txt")
	runpfile := filepath.Join(root, "Runpfile")
	spec := `
vars:
  d: sub
units:
  w:
    host:
      command: "pwd > ` + out + `"
      workdir: "{{vars runp_root}}/{{vars d}}"
`
	if err := os.WriteFile(runpfile, []byte(spec), 0644); err != nil {
		t.Fatal(err)
	}
	rf, err := LoadRunpfileFromPath(runpfile)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	// What "runp up --var d=other" does after loading.
	rf.Vars["d"] = "other"
	rf.Vars["runp_root"] = rf.Root
	sut := &RunpfileExecutor{rf: rf, LoggerFactory: createStubLogger, newPipe: os.Pipe}
	if err := startWithTimeout(t, sut, 10*time.Second); err != nil {
		t.Fatalf("start: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("unit did not run: %v", err)
	}
	got, _ := filepath.EvalSymlinks(strings.TrimSpace(string(data)))
	want, _ := filepath.EvalSymlinks(filepath.Join(root, "other"))
	if got != want {
		t.Errorf("workdir = %s, want %s", got, want)
	}
}

func TestHostStartCommandRejectsMissingWorkdir(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{Debug: false, Color: false})
	p := &HostProcess{CommandLine: "true", WorkingDir: filepath.Join(t.TempDir(), "missing")}
	_, err := p.StartCommand()
	if err == nil || !strings.Contains(err.Error(), "working directory") {
		t.Fatalf("expected working directory error, got %v", err)
	}
}
