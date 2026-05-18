package core

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// ErrReloadNotSupported is returned by StopUnit when the unit type cannot be
// stopped from a separate process. SSH tunnels run as goroutines inside
// runp up and have no external PID or socket to target.
var ErrReloadNotSupported = errors.New("unit type does not support reload from a separate process")

// StopUnit stops the running instance of a unit by probing external state.
// It works across process boundaries so it can be called from runp reload
// while runp up is running in another terminal.
//
// Returns ErrReloadNotSupported for ssh_tunnel units.
// Returns a non-fatal error (unit not running) when the process is already
// stopped; callers may choose to continue with the start phase.
func StopUnit(unit *RunpUnit, pidDir string, envSettings *EnvironmentSettings) error {
	timeout := parseStopTimeout(unit.StopTimeout)
	switch {
	case unit.Host != nil:
		return stopByPIDFile(unit.Name, timeout, pidDir)
	case unit.Container != nil:
		return stopContainer(unit.Container, envSettings)
	case unit.SSHTunnel != nil:
		return fmt.Errorf("unit %q: %w: SSH tunnels run inside the runp up process and cannot be reloaded externally", unit.Name, ErrReloadNotSupported)
	}
	return fmt.Errorf("unit %q has no known process type", unit.Name)
}

func parseStopTimeout(s string) time.Duration {
	if s != "" {
		if d, err := time.ParseDuration(s); err == nil {
			return d
		}
	}
	return 5 * time.Second
}

func stopByPIDFile(unitName string, timeout time.Duration, pidDir string) error {
	pid, err := ReadPIDFile(pidDir, unitName)
	if err != nil {
		return fmt.Errorf("unit %q does not appear to be running (no PID file found)", unitName)
	}
	if !isProcessAlive(pid) {
		RemovePIDFile(pidDir, unitName)
		return fmt.Errorf("unit %q: process %d has already exited", unitName, pid)
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		RemovePIDFile(pidDir, unitName)
		return fmt.Errorf("unit %q: process %d not found: %v", unitName, pid, err)
	}
	return stopProcessGracefully(proc, timeout)
}

func stopContainer(c *ContainerProcess, envSettings *EnvironmentSettings) error {
	runner, err := exec.LookPath(envSettings.ContainerRunnerExe)
	if err != nil {
		return fmt.Errorf("container runner not found: %s", envSettings.ContainerRunnerExe)
	}
	name := c.buildContainerName()
	out, err := exec.Command(runner, "stop", name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop container %q: %v (%s)", name, err, string(out))
	}
	return nil
}
