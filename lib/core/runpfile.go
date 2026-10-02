package core

import "fmt"

// Runpfile is the model containing the full configuration.
type Runpfile struct {
	Name          string
	Description   string
	Version       string
	Vars          map[string]string
	Root          string
	Units         map[string]*RunpUnit
	SecretKey     string `yaml:"-"`
	Include       []string
	Preconditions Preconditions
}

// RunpUnit is...
type RunpUnit struct {
	Name          string
	Description   string
	StopTimeout   string   `yaml:"stop_timeout"`
	DependsOn     []string `yaml:"depends_on"`
	Preconditions Preconditions
	Ready         ReadyCondition

	Host      *HostProcess
	Container *ContainerProcess
	SSHTunnel *SSHTunnelProcess `yaml:"ssh_tunnel"`

	vars                map[string]string
	secretKey           string
	root                string // root directory of the Runpfile defining the unit
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

// buildProcess returns the process of the unit. Vars in the working
// directory and in Env are not resolved here: the final vars (including --var
// overrides and runp_root) are only known when units are started, see
// resolveUnitWorkingDir and resolveEnvironment.
func (u *RunpUnit) buildProcess() RunpProcess {
	if u.Container != nil {
		return u.Container
	}
	if u.Host != nil {
		return u.Host
	}
	if u.SSHTunnel != nil {
		return u.SSHTunnel
	}
	return nil
}

// resolveUnitWorkingDir resolves vars in the working directory of the unit
// using its current vars. A relative result is made absolute against the root
// of the Runpfile defining the unit (containers keep the path as is: it lives
// inside the container).
func resolveUnitWorkingDir(u *RunpUnit, defaultRoot string) error {
	p := u.Process()
	if p == nil || !varsRegexp.MatchString(p.Dir()) {
		return nil
	}
	dir := newCliPreprocessor(u.vars).process(p.Dir())
	if !u.SkipDirResolution() {
		root := u.root
		if root == "" {
			root = defaultRoot
		}
		abs, err := resolvePath(dir, root)
		if err != nil {
			return fmt.Errorf("unit %s: cannot resolve working directory %s: %w", u.Name, dir, err)
		}
		dir = abs
	}
	p.SetDir(dir)
	return nil
}

// SkipDirResolution avoid resolve dir for containers
func (u *RunpUnit) SkipDirResolution() bool {
	return u.Container != nil
}
