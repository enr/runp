//go:build windows
// +build windows

package core

import "os"

// isProcessAlive returns true if the process with the given PID is running.
// On Windows, os.FindProcess calls OpenProcess and returns an error if the
// process does not exist or is not accessible.
func isProcessAlive(pid int) bool {
	_, err := os.FindProcess(pid)
	return err == nil
}
