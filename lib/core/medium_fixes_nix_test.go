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

// fakeContainerRunner writes a docker-like script that logs its arguments
// and answers ps/inspect with the given state and project label.
func fakeContainerRunner(t *testing.T, exists bool, state, project string) (runner, log string) {
	t.Helper()
	dir := t.TempDir()
	log = filepath.Join(dir, "calls.log")
	psOut := ""
	if exists {
		psOut = "abc123"
	}
	script := `#!/bin/sh
echo "$@" >> ` + log + `
case "$1" in
  ps) echo "` + psOut + `" ;;
  inspect)
    case "$3" in
      *State*) echo "` + state + `" ;;
      *Labels*) echo "` + project + `" ;;
    esac ;;
esac
exit 0
`
	runner = filepath.Join(dir, "fake-docker")
	if err := os.WriteFile(runner, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return runner, log
}

func TestContainerIsStartableExistingStoppedContainer(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{Debug: false, Color: false})
	runner, _ := fakeContainerRunner(t, true, "exited", "")
	cp := &ContainerProcess{Image: "alpine", environmentSettings: &EnvironmentSettings{ContainerRunnerExe: runner}}
	cp.SetID("web")
	ok, err := cp.IsStartable()
	if err != nil || ok {
		t.Fatalf("IsStartable = %v, %v; want false, nil", ok, err)
	}
	found := false
	for _, line := range testLogger.outputLines() {
		if strings.Contains(line, "already exists (status: exited)") {
			found = true
		}
	}
	if !found {
		t.Error("expected a message distinguishing an existing stopped container")
	}
}

func TestStopContainerRefusesOtherProject(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{Debug: false, Color: false})
	runner, log := fakeContainerRunner(t, true, "running", "other123")
	cp := &ContainerProcess{Image: "alpine", project: "mine4567"}
	cp.SetID("db")
	err := stopContainer(cp, 5*time.Second, &EnvironmentSettings{ContainerRunnerExe: runner})
	if err == nil || !strings.Contains(err.Error(), "another Runpfile") {
		t.Fatalf("expected refusal, got %v", err)
	}
	calls, _ := os.ReadFile(log)
	if strings.Contains(string(calls), "stop") {
		t.Errorf("container of another project was stopped: %s", calls)
	}
}

// Verifying preconditions (also done by --dry-run) must not create networks.
func TestContainerVerifyPreconditionsHasNoSideEffects(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{Debug: false, Color: false})
	runner, log := fakeContainerRunner(t, false, "", "")
	cp := &ContainerProcess{Image: "alpine", environmentSettings: &EnvironmentSettings{ContainerRunnerExe: runner}}
	if res := cp.VerifyPreconditions(); res.Vote != Proceed {
		t.Fatalf("unexpected vote %v: %v", res.Vote, res.Reasons)
	}
	if calls, _ := os.ReadFile(log); len(calls) != 0 {
		t.Errorf("VerifyPreconditions ran container commands: %s", calls)
	}
	if err := cp.PreStart(); err != nil {
		t.Fatalf("PreStart: %v", err)
	}
	if calls, _ := os.ReadFile(log); !strings.Contains(string(calls), "network create runp-network") {
		t.Errorf("PreStart did not create the network: %s", calls)
	}
}

// --dry-run must not connect to the SSH jump server to run test_command.
func TestDryRunDoesNotRunSSHTestCommand(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{Debug: false, Color: false})
	rf := &Runpfile{Units: map[string]*RunpUnit{
		"tunnel": {Name: "tunnel", SSHTunnel: &SSHTunnelProcess{
			User: "u", Auth: Auth{Secret: "s"}, InsecureIgnoreHostKey: true,
			Jump:        Endpoint{Host: "127.0.0.1", Port: 1},
			Target:      Endpoint{Host: "127.0.0.1", Port: 2},
			TestCommand: "true",
		}},
	}}
	previews := NewExecutor(rf).DryRunPreviews()
	if len(previews) != 1 || previews[0].Skipped {
		t.Errorf("dry-run evaluated the SSH test command: %+v", previews)
	}
}

// A unit waiting on await must stop waiting when another unit fails.
func TestAwaitIsCancelledOnAbort(t *testing.T) {
	sut := newLifetimeExecutor(t, `
units:
  waiting:
    host:
      command: "true"
      await:
        timeout: 30s
  pre:
    host:
      command: "sleep 0.3; exit 1"
  dependent:
    depends_on: [pre]
    host:
      command: "true"
`)
	start := time.Now()
	if err := startWithTimeout(t, sut, 10*time.Second); err == nil {
		t.Fatal("expected an error")
	}
	if elapsed := time.Since(start); elapsed < 300*time.Millisecond || elapsed > 5*time.Second {
		t.Errorf("await was not cancelled: Start took %v", elapsed)
	}
}

func TestPIDFilePermissions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "pids")
	if err := WritePIDFile(dir, "svc", 1234); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Stat(dir); fi.Mode().Perm() != 0700 {
		t.Errorf("pid dir mode = %v, want 0700", fi.Mode().Perm())
	}
	if fi, _ := os.Stat(filepath.Join(dir, "svc.pid")); fi.Mode().Perm() != 0600 {
		t.Errorf("pid file mode = %v, want 0600", fi.Mode().Perm())
	}
}
