package core

import (
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const containerNamePrefix = `runp-`

// ContainerProcess implements RunpProcess.
type ContainerProcess struct {
	// image
	Image string
	// if not used it will be created
	Name string
	// in format docker-compose
	Ports []string
	// rm Automatically remove the container when it exits
	SkipRm bool `yaml:"skip_rm"`
	// in format docker-compose
	Volumes     []string
	VolumesFrom []string `yaml:"volumes_from"`
	Mounts      []string
	ShmSize     string `yaml:"shm_size"`
	Command     string

	// generics
	WorkingDir string `yaml:"workdir"`
	Env        map[string]string
	Await      AwaitCondition

	id string
	// project identifies the Runpfile owning the container (see ProjectID);
	// it is set as the runp.project label.
	project             string
	vars                map[string]string
	preconditions       Preconditions
	secretKey           string
	stopTimeout         string
	environmentSettings *EnvironmentSettings
}

// ID for the sub process
func (p *ContainerProcess) ID() string {
	return p.id
}

// SetID for the sub process
func (p *ContainerProcess) SetID(id string) {
	p.id = id
}

// OnStarted implements RunpProcess. No-op for container processes.
func (p *ContainerProcess) OnStarted(_ int) {}

// PostStop implements RunpProcess. No-op for container processes.
func (p *ContainerProcess) PostStop() {}

// StartCommand returns the command starting the process.
func (p *ContainerProcess) StartCommand() (RunpCommand, error) {
	cmd, err := p.buildCmdImage()
	if err != nil {
		return nil, err
	}
	return &ExecCommandWrapper{
		cmd: cmd,
	}, nil
}

func (p *ContainerProcess) lookupContainerRunner() (string, error) {
	path, err := exec.LookPath(p.environmentSettings.ContainerRunnerExe)
	if err != nil {
		return "", fmt.Errorf("container runner %q not found in PATH: %w", p.environmentSettings.ContainerRunnerExe, err)
	}
	return path, nil
}

// StopCommand returns the command stopping the process.
func (p *ContainerProcess) StopCommand() (RunpCommand, error) {
	containerRunner, err := p.lookupContainerRunner()
	if err != nil {
		return nil, err
	}
	return &ExecCommandWrapper{
		cmd: exec.Command(containerRunner, containerStopArgs(p.buildContainerName(), p.StopTimeout())...),
	}, nil
}

// containerStopArgs returns the arguments of "<runner> stop", passing the
// unit's stop_timeout (rounded up to whole seconds) as the grace period.
func containerStopArgs(name string, timeout time.Duration) []string {
	secs := int((timeout + time.Second - 1) / time.Second)
	return []string{"stop", "-t", strconv.Itoa(secs), name}
}

// StopTimeout duration to wait to force kill process
func (p *ContainerProcess) StopTimeout() time.Duration {
	if p.stopTimeout != "" {
		d, err := time.ParseDuration(p.stopTimeout)
		if err != nil {
			return time.Duration(5) * time.Second
		}
		return d
	}
	return time.Duration(5) * time.Second
}

// Dir for the sub process
func (p *ContainerProcess) Dir() string {
	return p.WorkingDir
}

// SetDir for the sub process
func (p *ContainerProcess) SetDir(wd string) {
	p.WorkingDir = wd
}

func (p *ContainerProcess) buildContainerName() string {
	if p.Name != "" {
		return p.Name
	}
	return fmt.Sprintf("%s%s", containerNamePrefix, p.ID())
}

// buildArgs returns the argument list for "<runner> run ...". Each value is
// expanded exactly once and passed as a separate argument, without going
// through a shell, so values cannot inject commands on the host.
func (p *ContainerProcess) buildArgs() ([]string, error) {
	pre := newCliPreprocessor(p.vars)
	img := pre.process(p.Image)
	ui.Debugf("Run image '%s'\n", img)

	// rm Automatically remove the container when it exits
	args := []string{"run", "-t"}
	if !p.SkipRm {
		args = append(args, "--rm")
	}
	args = append(args, "--name", pre.process(p.buildContainerName()))
	if p.project != "" {
		args = append(args, "--label", containerProjectLabel+"="+p.project)
	}
	args = append(args, "--network", "runp-network")
	if p.ShmSize != "" {
		args = append(args, "--shm-size", pre.process(p.ShmSize))
	}
	for _, volume := range p.Volumes {
		args = append(args, "--volume", pre.process(volume))
	}
	for _, volume := range p.VolumesFrom {
		args = append(args, "--volumes-from", containerNamePrefix+pre.process(volume))
	}
	for _, m := range p.Mounts {
		args = append(args, "--mount", pre.process(m))
	}
	if p.WorkingDir != "" {
		args = append(args, "--workdir", pre.process(p.WorkingDir))
	}
	for _, ports := range p.Ports {
		args = append(args, "-p", pre.process(ports))
	}
	names := make([]string, 0, len(p.Env))
	for name := range p.Env {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		args = append(args, "-e", name+"="+expandEnvValue(pre.process(p.Env[name])))
	}
	args = append(args, img)
	if p.Command != "" {
		words, err := splitCommandLine(pre.process(p.Command))
		if err != nil {
			return nil, fmt.Errorf("invalid command for container %s: %w", p.ID(), err)
		}
		args = append(args, words...)
	}
	return args, nil
}

func (p *ContainerProcess) buildCmdImage() (*exec.Cmd, error) {
	containerRunner, err := p.lookupContainerRunner()
	if err != nil {
		return nil, err
	}
	args, err := p.buildArgs()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(containerRunner, args...)
	ui.Debugf("Container command:\n%s", strings.Join(cmd.Args, " "))
	return cmd, nil
}

// ShouldWait returns if the process has await set.
func (p *ContainerProcess) ShouldWait() bool {
	return (p.Await.Timeout != "")
}

// AwaitResource returns the await resource.
func (p *ContainerProcess) AwaitResource() string {
	return p.Await.Resource
}

// AwaitTimeout returns the await timeout.
func (p *ContainerProcess) AwaitTimeout() string {
	return p.Await.Timeout
}

// String representation of process
func (p *ContainerProcess) String() string {
	return fmt.Sprintf("%T{id=%s container=%s}", p, p.ID(), p.buildContainerName())
}

// IsStartable checks that no container with the same name exists.
func (p *ContainerProcess) IsStartable() (bool, error) {
	containerRunner, err := p.lookupContainerRunner()
	if err != nil {
		return false, err
	}
	cn := p.buildContainerName()
	state, exists, err := containerState(containerRunner, cn)
	if err != nil {
		return false, err
	}
	if !exists {
		return true, nil
	}
	if state == "running" {
		ui.WriteLinef("Container %s cannot be started: a container with this name is already running", cn)
	} else {
		ui.WriteLinef("Container %s cannot be started: a container with this name already exists (status: %s); remove it with: %s rm %s", cn, state, containerRunner, cn)
	}
	if owner, err := containerProject(containerRunner, cn); err == nil && owner != "" && p.project != "" && owner != p.project {
		ui.WriteLinef("Container %s belongs to another Runpfile: set a different container name in the unit", cn)
	}
	return false, nil
}

const containerProjectLabel = "runp.project"

// containerNameFilter returns a --filter value matching exactly name. Docker
// matches the filter against names with a leading slash, podman without.
func containerNameFilter(name string) string {
	return "name=^/?" + regexp.QuoteMeta(name) + "$"
}

// containerState returns the status of the container named name (e.g.
// "running", "exited") and whether it exists.
func containerState(runner, name string) (string, bool, error) {
	cmd := exec.Command(runner, "ps", "-aq", "--filter", containerNameFilter(name))
	ui.Debugf("Container lookup command:\n%s", strings.Join(cmd.Args, " "))
	out, err := cmd.Output()
	if err != nil {
		return "", false, err
	}
	if strings.TrimSpace(string(out)) == "" {
		return "", false, nil
	}
	out, err = exec.Command(runner, "inspect", "--format", "{{.State.Status}}", name).Output()
	if err != nil {
		return "unknown", true, nil
	}
	return strings.TrimSpace(string(out)), true, nil
}

// containerProject returns the runp.project label of the container named
// name, or "" if it has none (e.g. not started by runp).
func containerProject(runner, name string) (string, error) {
	out, err := exec.Command(runner, "inspect", "--format", `{{index .Config.Labels "`+containerProjectLabel+`"}}`, name).Output()
	if err != nil {
		return "", err
	}
	label := strings.TrimSpace(string(out))
	if label == "<no value>" {
		return "", nil
	}
	return label, nil
}

// SetPreconditions set preconditions.
func (p *ContainerProcess) SetPreconditions(preconditions Preconditions) {
	p.preconditions = preconditions
}

// VerifyPreconditions checks if the process can be started. It has no side
// effects (it is also used by --dry-run): the container network is created
// by PreStart.
func (p *ContainerProcess) VerifyPreconditions() PreconditionVerifyResult {
	res := p.preconditions.Verify()
	if res.Vote != Proceed {
		return res
	}
	if _, err := p.lookupContainerRunner(); err != nil {
		return PreconditionVerifyResult{
			Vote:    Stop,
			Reasons: []string{err.Error()},
		}
	}
	return PreconditionVerifyResult{
		Vote:    Proceed,
		Reasons: []string{},
	}
}

// PreStart creates the runp-network container network if it does not exist.
func (p *ContainerProcess) PreStart() error {
	containerRunner, err := p.lookupContainerRunner()
	if err != nil {
		return err
	}
	exists, err := containerNetworkExists(containerRunner)
	if err != nil || exists {
		return err
	}
	command := exec.Command(containerRunner, "network", "create", "runp-network")
	cmdLine := strings.Join(command.Args, " ")
	ui.Debugf("Creating network: %s", cmdLine)
	if out, err := command.CombinedOutput(); err != nil {
		// Another container unit starting concurrently may have created it.
		if exists, _ := containerNetworkExists(containerRunner); exists {
			return nil
		}
		return fmt.Errorf("failed to create network: %s (%v): %s", cmdLine, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func containerNetworkExists(containerRunner string) (bool, error) {
	command := exec.Command(containerRunner, "network", "ls", "--filter", "name=runp-network", "--format", "{{ .Name }}")
	cmdLine := strings.Join(command.Args, " ")
	ui.Debugf("Checking network: %s", cmdLine)
	out, err := command.Output()
	if err != nil {
		return false, fmt.Errorf("failed to read network check command output: %s (%v)", cmdLine, err)
	}
	for _, name := range strings.Fields(string(out)) {
		if name == "runp-network" {
			return true, nil
		}
	}
	return false, nil
}
