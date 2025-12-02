//go:build darwin || freebsd || linux || netbsd || openbsd
// +build darwin freebsd linux netbsd openbsd

package core

import (
	"os/exec"
	"syscall"
)

// configureProcessAttributes configures process attributes for Unix systems.
// Sets Setpgid to true so the process runs in its own process group.
// This allows us to send signals to the entire process group without affecting runp.
func configureProcessAttributes(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	// Set process group ID to new pid (SYSV setpgrp)
	// This puts the process in its own process group, allowing us to send
	// signals to the entire group (bash and its children) without affecting runp
	cmd.SysProcAttr.Setpgid = true
}
