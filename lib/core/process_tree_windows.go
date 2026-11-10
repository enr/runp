//go:build windows
// +build windows

package core

import (
	"os"
	"os/exec"
	"strconv"
)

// killProcessTree forcibly terminates p and all its descendants.
// os.Process.Kill (TerminateProcess) stops p alone: the children of a
// "cmd /C" shell would survive it, e.g. keeping a port open.
func killProcessTree(p *os.Process) error {
	if err := taskkill(p.Pid, true); err == nil {
		return nil
	}
	// taskkill unavailable or failed: terminate at least the process itself.
	return p.Kill()
}

// taskkill runs "taskkill /T /PID pid", with /F when force is set. Without /F
// the processes are only asked to close: windowed applications honour it,
// console programs refuse it and taskkill fails.
func taskkill(pid int, force bool) error {
	args := []string{"/T", "/PID", strconv.Itoa(pid)}
	if force {
		args = append([]string{"/F"}, args...)
	}
	return exec.Command("taskkill", args...).Run()
}
