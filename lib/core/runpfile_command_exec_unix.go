//go:build darwin || freebsd || linux || netbsd || openbsd
// +build darwin freebsd linux netbsd openbsd

package core

import (
	"os"
	"syscall"
	"time"
)

// stopProcess implements graceful shutdown for Unix systems: it sends SIGTERM
// to the process group (so bash trap functions and child processes get it),
// waits up to timeout, then sends SIGKILL.
//
// exited reports whether the owner of the process has already reaped it; the
// process is never waited on here, the executor owns the Wait() call.
func stopProcess(p *os.Process, exited func() bool, timeout time.Duration, id string) error {
	if p == nil || exited() {
		return nil
	}

	if !signalGroup(p, syscall.SIGTERM) {
		// Process might have already exited
		return nil
	}

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		<-ticker.C
		if exited() {
			return nil
		}
		// Signal 0 checks existence without delivering a signal.
		if err := p.Signal(syscall.Signal(0)); err != nil {
			return nil
		}
	}

	if id != "" {
		ui.Debugf("Process %s did not terminate gracefully within %v, forcing kill", id, timeout)
	}
	if !exited() {
		signalGroup(p, syscall.SIGKILL)
	}
	return nil
}

// signalGroup sends sig to the process group led by p (host units are started
// with Setpgid), falling back to p alone. The group is signalled only when p
// is its leader: a process sharing the caller's group must never cause runp
// itself to be signalled. It returns false when the process no longer exists.
func signalGroup(p *os.Process, sig syscall.Signal) bool {
	if pgid, err := syscall.Getpgid(p.Pid); err == nil && pgid == p.Pid {
		if err := syscall.Kill(-pgid, sig); err == nil {
			return true
		}
	}
	return p.Signal(sig) == nil
}
