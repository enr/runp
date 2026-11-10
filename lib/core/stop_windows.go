//go:build windows
// +build windows

package core

import (
	"os"
	"time"
)

// stopProcessGracefully asks the process tree of proc to close and waits up to
// timeout, then terminates the tree if it is still alive.
//
// os.Process.Signal supports only os.Kill on Windows (os.Interrupt always
// fails), and runp reload runs in another console than the unit, so Ctrl+C
// cannot be delivered either: a non-forced taskkill is the only graceful
// request available. Console programs refuse it and are terminated at once.
func stopProcessGracefully(proc *os.Process, timeout time.Duration) error {
	if err := taskkill(proc.Pid, false); err == nil {
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
	}
	if !isProcessAlive(proc.Pid) {
		return nil
	}
	return killProcessTree(proc)
}
