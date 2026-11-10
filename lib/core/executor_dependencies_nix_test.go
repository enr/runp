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

// A dependency that exits successfully must release its dependents even when
// a background child it started keeps its output open.
func TestStart_BackgroundChildDoesNotBlockDependents(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "dependent-ran")
	sut := newLifetimeExecutor(t, `
units:
  init:
    host:
      command: "(sleep 5) & echo daemon started"
  app:
    depends_on: [init]
    host:
      command: "touch `+marker+`"
`)
	start := time.Now()
	go sut.Start()
	for time.Since(start) < 3*time.Second {
		if _, err := os.Stat(marker); err == nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("dependent not started while the dependency's background child was running")
}

// A unit that exits with an error must make Start fail even when no other
// unit depends on it, so that runp exits with a non-zero code.
func TestStart_UnitExitErrorFailsStart(t *testing.T) {
	notShuttingDown(t)
	sut := newLifetimeExecutor(t, `
units:
  bad:
    host:
      command: "exit 7"
  good:
    host:
      command: "true"
`)
	err := startWithTimeout(t, sut, 10*time.Second)
	if err == nil {
		t.Fatal("expected an error when a unit exits with an error")
	}
	if !strings.Contains(err.Error(), "exited with an error") {
		t.Errorf("unexpected error: %v", err)
	}
}

// notShuttingDown clears the shutting-down flag of the global application
// context for the duration of the test: other tests may leave it set, and
// exits during a shutdown are not reported as errors.
func notShuttingDown(t *testing.T) {
	t.Helper()
	ctx := GetApplicationContext()
	ctx.Lock()
	was := ctx.shuttingDown
	ctx.shuttingDown = false
	ctx.Unlock()
	t.Cleanup(func() {
		ctx.Lock()
		ctx.shuttingDown = was
		ctx.Unlock()
	})
}

// A unit whose process dies before its ready.delay or ready.cmd condition is
// satisfied is not ready: its dependents must not start.
func TestStart_ProcessDeathFailsReadiness(t *testing.T) {
	for name, ready := range map[string]string{
		"delay": "delay: 3s",
		"cmd":   "cmd: \"false\"",
	} {
		t.Run(name, func(t *testing.T) {
			notShuttingDown(t)
			marker := filepath.Join(t.TempDir(), "dependent-ran")
			sut := newLifetimeExecutor(t, `
units:
  db:
    ready:
      `+ready+`
      timeout: 5s
    host:
      command: "exit 3"
  app:
    depends_on: [db]
    host:
      command: "touch `+marker+`"
`)
			start := time.Now()
			err := startWithTimeout(t, sut, 10*time.Second)
			if err == nil {
				t.Error("expected an error when a dependency dies before being ready")
			}
			if _, statErr := os.Stat(marker); statErr == nil {
				t.Error("dependent started although its dependency died before being ready")
			}
			if elapsed := time.Since(start); elapsed > 2*time.Second {
				t.Errorf("readiness failure detected after %v, expected as soon as the process exited", elapsed)
			}
		})
	}
}

// A process that exits successfully before its ready condition is satisfied
// has completed its work: it is ready, like a unit without ready condition.
func TestStart_ProcessSuccessfulExitIsReady(t *testing.T) {
	notShuttingDown(t)
	marker := filepath.Join(t.TempDir(), "dependent-ran")
	sut := newLifetimeExecutor(t, `
units:
  migrate:
    ready:
      delay: 3s
    host:
      command: "true"
  app:
    depends_on: [migrate]
    host:
      command: "touch `+marker+`"
`)
	if err := startWithTimeout(t, sut, 10*time.Second); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(marker); statErr != nil {
		t.Error("dependent not started although its dependency completed successfully")
	}
}

// A unit whose ready condition fails is a failure of the run even when no
// other unit depends on it: runp stops the running units and reports it.
func TestStart_FailedReadinessWithoutDependentsFailsStart(t *testing.T) {
	notShuttingDown(t)
	sut := newLifetimeExecutor(t, `
units:
  svc:
    ready:
      output: "never printed"
      timeout: 500ms
    host:
      command: "sleep 30"
  other:
    host:
      command: "sleep 30"
`)
	start := time.Now()
	err := startWithTimeout(t, sut, 15*time.Second)
	if err == nil {
		t.Error("expected an error when a ready condition fails")
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Errorf("running units not stopped after the readiness failure (took %v)", elapsed)
	}
}

// {{vars}} in the ready and await blocks and in host.executable are expanded.
func TestStart_VarsExpandedInReadyAwaitAndExecutable(t *testing.T) {
	notShuttingDown(t)
	dir := t.TempDir()
	marker := filepath.Join(dir, "dependent-ran")
	sut := newLifetimeExecutor(t, `
units:
  svc:
    ready:
      cmd: 'test "{{vars msg}}" = hello'
      timeout: "{{vars timeout}}"
    host:
      executable: "{{vars sh}}"
      args: ["-c", "sleep 1"]
      await:
        resource: "file://{{vars dir}}"
        timeout: "{{vars timeout}}"
  dependent:
    depends_on: [svc]
    host:
      command: "touch `+marker+`"
`)
	sut.rf.Vars = map[string]string{"msg": "hello", "timeout": "5s", "sh": "sh", "dir": dir}
	if err := startWithTimeout(t, sut, 15*time.Second); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(marker); statErr != nil {
		t.Error("dependent not started: vars not expanded in ready/await/executable")
	}
}
