//go:build windows
// +build windows

package core

import (
	"os"
	"strconv"
	"syscall"
)

// isProcessAlive returns true if the process with the given PID is running.
// On Windows, os.FindProcess calls OpenProcess and returns an error if the
// process does not exist or is not accessible.
func isProcessAlive(pid int) bool {
	_, err := os.FindProcess(pid)
	return err == nil
}

const processQueryLimitedInformation = 0x1000

// processIdentity returns the creation time of the process with the given
// PID, or "" if it cannot be determined.
func processIdentity(pid int) string {
	h, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return ""
	}
	defer syscall.CloseHandle(h)
	var creation, exit, kernel, user syscall.Filetime
	if err := syscall.GetProcessTimes(h, &creation, &exit, &kernel, &user); err != nil {
		return ""
	}
	return strconv.FormatInt(creation.Nanoseconds(), 10)
}
