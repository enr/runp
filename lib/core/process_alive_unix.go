//go:build darwin || freebsd || linux || netbsd || openbsd
// +build darwin freebsd linux netbsd openbsd

package core

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// isProcessAlive returns true if the process with the given PID is running.
// Uses signal 0, which checks process existence without delivering a signal.
func isProcessAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

// processIdentity returns a value identifying the process instance with the
// given PID (its start time), or "" if it cannot be determined. Two processes
// that happen to get the same PID have different identities.
func processIdentity(pid int) string {
	// Linux: field 22 of /proc/<pid>/stat is the start time in clock ticks.
	if data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat"); err == nil {
		s := string(data)
		// The command name (field 2) may contain spaces: skip past it.
		if i := strings.LastIndexByte(s, ')'); i >= 0 {
			fields := strings.Fields(s[i+1:])
			// fields[0] is field 3 (state), so field 22 is fields[19].
			if len(fields) > 19 {
				return fields[19]
			}
		}
	}
	out, err := exec.Command("ps", "-o", "lstart=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return ""
	}
	return strings.Join(strings.Fields(string(out)), " ")
}
