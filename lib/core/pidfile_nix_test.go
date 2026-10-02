//go:build darwin || freebsd || linux || netbsd || openbsd
// +build darwin freebsd linux netbsd openbsd

package core

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func startGroupLeader(t *testing.T, script string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command("bash", "-c", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd
}

// A PID file whose PID now belongs to a different process must not cause
// that process to be signalled.
func TestStopByPIDFile_StalePIDIsNotSignalled(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{Debug: false, Color: false})
	cmd := startGroupLeader(t, "sleep 5")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cmd.Process.Kill()
		cmd.Wait()
	}()
	dir := t.TempDir()
	// Simulate a stale file: same PID, identity of another process.
	content := strconv.Itoa(cmd.Process.Pid) + "\nnot-this-process\n"
	if err := os.WriteFile(filepath.Join(dir, "svc.pid"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	err := stopByPIDFile("svc", time.Second, dir)
	if err == nil || !strings.Contains(err.Error(), "stale") {
		t.Errorf("expected stale PID error, got %v", err)
	}
	if !isProcessAlive(cmd.Process.Pid) {
		t.Error("an unrelated process was signalled")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "svc.pid")); statErr == nil {
		t.Error("stale PID file should be removed")
	}
}

// Stopping through the PID file must stop the children of the unit's shell.
func TestStopByPIDFile_StopsProcessGroup(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{Debug: false, Color: false})
	cmd := startGroupLeader(t, "sleep 30 & echo $!; wait")
	stdout, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	go cmd.Wait()
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	child, _ := strconv.Atoi(strings.TrimSpace(line))
	defer syscall.Kill(child, syscall.SIGKILL)

	dir := t.TempDir()
	if err := WritePIDFile(dir, "svc", cmd.Process.Pid); err != nil {
		t.Fatal(err)
	}
	if err := stopByPIDFile("svc", 2*time.Second, dir); err != nil {
		t.Fatalf("stop: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for isProcessAlive(child) {
		if time.Now().After(deadline) {
			t.Fatal("child of the unit's shell survived the stop")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestRemovePIDFileIfOwned(t *testing.T) {
	dir := t.TempDir()
	if err := WritePIDFile(dir, "svc", 4242); err != nil {
		t.Fatal(err)
	}
	removePIDFileIfOwned(dir, "svc", 1111)
	if pid, err := ReadPIDFile(dir, "svc"); err != nil || pid != 4242 {
		t.Fatalf("PID file of another instance was removed: pid=%d err=%v", pid, err)
	}
	removePIDFileIfOwned(dir, "svc", 4242)
	if _, err := ReadPIDFile(dir, "svc"); err == nil {
		t.Error("own PID file should be removed")
	}
}

func TestProcessIdentity(t *testing.T) {
	id := processIdentity(os.Getpid())
	if id == "" {
		t.Skip("process identity not available on this system")
	}
	if id != processIdentity(os.Getpid()) {
		t.Error("identity of the same process must be stable")
	}
}
