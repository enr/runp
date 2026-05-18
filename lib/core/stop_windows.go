//go:build windows
// +build windows

package core

import (
	"os"
	"time"
)

// stopProcessGracefully sends os.Interrupt (Ctrl+C) and waits up to timeout,
// then calls Kill() if the process is still alive.
func stopProcessGracefully(proc *os.Process, timeout time.Duration) error {
	if err := proc.Signal(os.Interrupt); err != nil {
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

	ui.Debugf("Process %d did not stop within %v, killing", proc.Pid, timeout)
	_ = proc.Kill()
	return nil
}
