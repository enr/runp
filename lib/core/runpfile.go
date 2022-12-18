package core

import "fmt"

// Runpfile is the model containing the full configuration.
type Runpfile struct {
	Name        string
	Description string
	Version     string
	Vars        map[string]string
	Root        string
	Units       map[string]*RunpUnit
	SecretKey   string `yaml:"-"`
	Include     []string
}

// RunpUnit is...
type RunpUnit struct {
	Name          string
	Description   string
	StopTimeout   string `yaml:"stop_timeout"`
	Preconditions Preconditions

	Host      *HostProcess
	Container *ContainerProcess
	SSHTunnel *SSHTunnelProcess `yaml:"ssh_tunnel"`

	vars                map[string]string
	secretKey           string
	process             RunpProcess
	environmentSettings *EnvironmentSettings
}

// Process returns the sub process
func (u *RunpUnit) Process() RunpProcess {
	if u.process == nil {
		p := u.buildProcess()
		u.process = p
	}
	return u.process
}

// Kind describes the unit in `runp ls`.
func (u *RunpUnit) Kind() string {
	if u.Container != nil {
		return fmt.Sprintf(`Container process %s`, u.Container.Image)
	}
	if u.Host != nil {
		return `Host process`
	}
	if u.SSHTunnel != nil {
		st := u.SSHTunnel
		return fmt.Sprintf(`SSH tunnel %s -> %s -> %s`, st.Local.String(), st.Jump.String(), st.Target.String())
	}
	return ``
}

func (u *RunpUnit) buildProcess() RunpProcess {
	cliPreprocessor := newCliPreprocessor(u.vars)
	if u.Container != nil {
		container := u.Container
		container.WorkingDir = cliPreprocessor.process(u.Container.WorkingDir)
		return container
	}
	if u.Host != nil {
		host := u.Host
		host.WorkingDir = cliPreprocessor.process(u.Host.WorkingDir)
		return host
	}
	if u.SSHTunnel != nil {
		tunnel := u.SSHTunnel
		tunnel.WorkingDir = cliPreprocessor.process(u.SSHTunnel.WorkingDir)
		return tunnel
	}
	return nil
}

// SkipDirResolution avoid resolve dir for containers
func (u *RunpUnit) SkipDirResolution() bool {
	return u.Container != nil
}
