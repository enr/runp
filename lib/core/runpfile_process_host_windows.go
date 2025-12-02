//go:build windows
// +build windows

package core

import (
	"os/exec"
	"syscall"
)

// configureProcessAttributes configures process attributes for Windows.
// Sets CREATE_NEW_PROCESS_GROUP to true so the process runs in its own process group.
func configureProcessAttributes(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	// Create a new process group for the child process
	// This allows us to manage the process group independently
	cmd.SysProcAttr.CreationFlags = syscall.CREATE_NEW_PROCESS_GROUP
}
