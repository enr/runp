//go:build darwin || freebsd || linux || netbsd || openbsd
// +build darwin freebsd linux netbsd openbsd

package core

import (
	"os"
	"time"
)

// stopProcessGracefully stops proc, a process started by another runp
// instance (runp reload): SIGTERM to its process group (host units run in
// their own group, so children of the shell are stopped too), then SIGKILL
// to the processes of the group still alive after timeout.
func stopProcessGracefully(proc *os.Process, timeout time.Duration) error {
	// The process is not a child of this runp: it is never reaped here.
	notReaped := func() bool { return false }
	return stopProcess(proc, notReaped, timeout, "")
}
