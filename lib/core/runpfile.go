package core

// Runpfile is the model containing the full configuration.
type Runpfile struct {
	Name        string
	Description string
	Version     string
	Vars        map[string]string
	Root        string
	Units       map[string]*RunpUnit
	SecretKey   string `yaml:"-"`
}

// RunpUnit is...
type RunpUnit struct {
	Name          string
	Description   string
	StopTimeout   string `yaml:"stop_timeout"`
	Preconditions []map[string]interface{}

	Host      *HostProcess
	Container *ContainerProcess
	SSHTunnel *SSHTunnelProcess `yaml:"ssh_tunnel"`

	vars      map[string]string
	secretKey string
	process   RunpProcess
}

// Preconditions.
func (u *RunpUnit) ToPreconditions() []Precondition {
	return []Precondition{}
}

// Process for the sub process
func (u *RunpUnit) Process() RunpProcess {
	if u.process == nil {
		u.process = u.buildProcess()
	}
	return u.process
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
