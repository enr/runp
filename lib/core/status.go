package core

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"
)

// UnitState represents the observable runtime state of a unit.
type UnitState string

const (
	UnitStateRunning UnitState = "running"
	UnitStateStopped UnitState = "stopped"
	UnitStateUnknown UnitState = "unknown"
)

// UnitStatus holds the probed runtime status of a single unit.
type UnitStatus struct {
	Name   string
	Kind   string
	State  UnitState
	Detail string
}

// ProbeStatus determines the runtime status of a unit by inspecting external
// state (PID files, container engine, TCP ports). It works across process
// boundaries, so it can be called from a separate runp invocation while
// runp up is running in another terminal.
func ProbeStatus(unit *RunpUnit, pidDir string, envSettings *EnvironmentSettings) UnitStatus {
	s := UnitStatus{
		Name: unit.Name,
		Kind: unitKindLabel(unit),
	}
	switch {
	case unit.Host != nil:
		s.State, s.Detail = probeHostStatus(pidDir, unit.Name)
	case unit.Container != nil:
		s.State, s.Detail = probeContainerStatus(unit.Container, envSettings)
	case unit.SSHTunnel != nil:
		s.State, s.Detail = probeSSHTunnelStatus(unit.SSHTunnel)
	default:
		s.State = UnitStateUnknown
		s.Detail = "no process type defined"
	}
	return s
}

func unitKindLabel(unit *RunpUnit) string {
	switch {
	case unit.Container != nil:
		return "container"
	case unit.Host != nil:
		return "host"
	case unit.SSHTunnel != nil:
		return "ssh_tunnel"
	default:
		return "unknown"
	}
}

func probeHostStatus(pidDir, unitName string) (UnitState, string) {
	pid, err := ReadPIDFile(pidDir, unitName)
	if err != nil {
		return UnitStateUnknown, "no pid file"
	}
	if isProcessAlive(pid) {
		return UnitStateRunning, fmt.Sprintf("pid %d", pid)
	}
	return UnitStateStopped, fmt.Sprintf("pid %d exited", pid)
}

func probeContainerStatus(c *ContainerProcess, envSettings *EnvironmentSettings) (UnitState, string) {
	runner, err := exec.LookPath(envSettings.ContainerRunnerExe)
	if err != nil {
		return UnitStateUnknown, fmt.Sprintf("container runner not found: %s", envSettings.ContainerRunnerExe)
	}
	name := c.buildContainerName()
	// Anchor the filter with ^ and $ to avoid partial-name matches.
	out, err := exec.Command(runner, "ps", "--filter", "name=^"+name+"$", "--format", "{{.Status}}").Output()
	if err != nil {
		return UnitStateUnknown, fmt.Sprintf("container query failed: %v", err)
	}
	status := strings.TrimSpace(string(out))
	if status == "" {
		return UnitStateStopped, "container not running"
	}
	return UnitStateRunning, status
}

func probeSSHTunnelStatus(t *SSHTunnelProcess) (UnitState, string) {
	host := t.Local.Host
	if host == "" {
		host = "localhost"
	}
	addr := fmt.Sprintf("%s:%d", host, t.Local.Port)
	conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
	if err != nil {
		return UnitStateStopped, fmt.Sprintf("port %d not listening", t.Local.Port)
	}
	conn.Close()
	return UnitStateRunning, fmt.Sprintf("port %d listening", t.Local.Port)
}
