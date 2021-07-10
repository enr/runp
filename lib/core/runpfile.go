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
	Name        string
	Description string
	StopTimeout string `yaml:"stop_timeout"`
	Host        *HostProcess
	Container   *ContainerProcess
	SSHTunnel   *SSHTunnelProcess `yaml:"ssh_tunnel"`

	vars      map[string]string
	secretKey string
	//cliPreprocessor *cliPreprocessor
}

// Process for the sub process
func (u *RunpUnit) Process() RunpProcess {
	if u.Container != nil {
		// u.Container.vars = u.vars
		// u.Container.secretKey = u.secretKey
		// u.Container.stopTimeout = u.StopTimeout
		return u.Container
	}
	if u.Host != nil {
		// u.Host.vars = u.vars
		// u.Host.secretKey = u.secretKey
		// u.Host.stopTimeout = u.StopTimeout
		return u.Host
	}
	if u.SSHTunnel != nil {
		// u.SSHTunnel.vars = u.vars
		// u.SSHTunnel.secretKey = u.secretKey
		// u.SSHTunnel.stopTimeout = u.StopTimeout
		return u.SSHTunnel
	}
	return nil
}

// SkipDirResolution avoid resolve dir for containers
func (u *RunpUnit) SkipDirResolution() bool {
	return u.Container != nil
}
