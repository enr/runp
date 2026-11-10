//go:build windows
// +build windows

package core

import (
	"io"
	"os"
	"os/exec"
	"testing"
	"time"
)

// startShellTree starts "cmd /C ping ..." (a shell with a long-running child)
// with its output on a pipe, and returns the command and the read end. The
// pipe reaches EOF only when both the shell and its child have exited.
func startShellTree(t *testing.T) (*exec.Cmd, *os.File) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	cmd := exec.Command("cmd", "/C", "ping -n 60 127.0.0.1")
	configureProcessAttributes(cmd)
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	w.Close()
	t.Cleanup(func() {
		taskkill(cmd.Process.Pid, true)
		r.Close()
	})
	// Let the shell start its child.
	time.Sleep(time.Second)
	return cmd, r
}

// outputClosedWithin reports whether r reaches EOF within timeout.
func outputClosedWithin(r *os.File, timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		io.Copy(io.Discard, r)
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// runp reload stops a host unit with stopProcessGracefully: it must stop the
// shell and its children (os.Interrupt is not supported on Windows).
func TestStopProcessGracefullyStopsProcessTree(t *testing.T) {
	cmd, r := startShellTree(t)
	go cmd.Wait()
	if err := stopProcessGracefully(cmd.Process, 2*time.Second); err != nil {
		t.Logf("stopProcessGracefully: %v", err)
	}
	if !outputClosedWithin(r, 15*time.Second) {
		t.Error("process tree still running after stopProcessGracefully")
	}
}

// runp up stops a host unit with stopProcess: it must stop the shell and its
// children, not only the shell.
func TestStopProcessStopsProcessTree(t *testing.T) {
	cmd, r := startShellTree(t)
	exited := make(chan struct{})
	go func() {
		cmd.Wait()
		close(exited)
	}()
	hasExited := func() bool {
		select {
		case <-exited:
			return true
		default:
			return false
		}
	}
	if err := stopProcess(cmd.Process, hasExited, 2*time.Second, "tree"); err != nil {
		t.Logf("stopProcess: %v", err)
	}
	if !outputClosedWithin(r, 15*time.Second) {
		t.Error("children of the shell still running after stopProcess")
	}
}
