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
		// Don't process Env here - it will be processed at runtime in resolveEnvironment()
		// container.Env = processEnv(container.Env, cliPreprocessor)
		return container
	}
	if u.Host != nil {
		host := u.Host
		host.WorkingDir = cliPreprocessor.process(u.Host.WorkingDir)
		// Don't process Env here - it will be processed at runtime in resolveEnvironment()
		// host.Env = processEnv(host.Env, cliPreprocessor)
		return host
	}
	if u.SSHTunnel != nil {
		tunnel := u.SSHTunnel
		tunnel.WorkingDir = cliPreprocessor.process(u.SSHTunnel.WorkingDir)
		// Don't process Env here - it will be processed at runtime in resolveEnvironment()
		// tunnel.Env = processEnv(tunnel.Env, cliPreprocessor)
		return tunnel
	}
	return nil
}

func processEnv(env map[string]string, cliPreprocessor *cliPreprocessor) map[string]string {
	envmap := map[string]string{}
	for k, v := range env {
		envmap[k] = cliPreprocessor.process(v)
	}
	return envmap
}

// SkipDirResolution avoid resolve dir for containers
func (u *RunpUnit) SkipDirResolution() bool {
	return u.Container != nil
}
