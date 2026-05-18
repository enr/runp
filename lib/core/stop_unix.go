//go:build darwin || freebsd || linux || netbsd || openbsd
// +build darwin freebsd linux netbsd openbsd

package core

import (
	"os"
	"syscall"
	"time"
)

// stopProcessGracefully sends SIGTERM and waits up to timeout for the process
// to exit, then sends SIGKILL if it is still alive.
func stopProcessGracefully(proc *os.Process, timeout time.Duration) error {
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		// Process may have already exited between the isProcessAlive check and here.
		return nil
	}

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		<-ticker.C
		if !isProcessAlive(proc.Pid) {
			return nil
		}
	}

	ui.Debugf("Process %d did not stop within %v, sending SIGKILL", proc.Pid, timeout)
	_ = proc.Signal(syscall.SIGKILL)
	return nil
}
