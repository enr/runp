package core

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
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

	id                  string
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

// PreStart implements RunpProcess. No-op for container processes.
func (p *ContainerProcess) PreStart() error { return nil }

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
		cmd: exec.Command(containerRunner, "stop", p.buildContainerName()),
	}, nil
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
		args = append(args, "-e", name+"="+os.ExpandEnv(pre.process(p.Env[name])))
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

// IsStartable ...
func (p *ContainerProcess) IsStartable() (bool, error) {
	containerRunner, err := p.lookupContainerRunner()
	if err != nil {
		return false, err
	}
	cn := p.buildContainerName()
	cmd := exec.Command(containerRunner, "ps", "-aq", "-f", "name="+cn)
	ui.Debugf("IsStartable command:\n%s", strings.Join(cmd.Args, " "))
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	so := string(out)
	ui.Debugf("Container startability check output: %s", so)
	if so != "" {
		ui.WriteLinef("Container %s cannot be started: container is already running (output: %s)", cn, so)
		return false, nil
	}
	return true, nil
}

// SetPreconditions set preconditions.
func (p *ContainerProcess) SetPreconditions(preconditions Preconditions) {
	p.preconditions = preconditions
}

// VerifyPreconditions check if process can be started
func (p *ContainerProcess) VerifyPreconditions() PreconditionVerifyResult {

	res := p.preconditions.Verify()
	if res.Vote != Proceed {
		return res
	}
	containerRunner, err := p.lookupContainerRunner()
	if err != nil {
		return PreconditionVerifyResult{
			Vote:    Stop,
			Reasons: []string{err.Error()},
		}
	}
	command := exec.Command(containerRunner, "network", "ls", "--filter", "name=runp-network", "--format", "{{ .Name }}")
	cmdLine := strings.Join(command.Args, " ")
	ui.Debugf("Checking network precondition: %s", cmdLine)
	out, err := command.Output()
	if err != nil {
		return PreconditionVerifyResult{
			Vote:    Stop,
			Reasons: []string{fmt.Sprintf("Failed to read network check command output: %s (%v)", cmdLine, err)},
		}
	}
	so := strings.TrimSpace(string(out))
	ui.Debugf("Network check output: %s", so)
	if so == "runp-network" {
		return PreconditionVerifyResult{
			Vote:    Proceed,
			Reasons: []string{},
		}
	}
	command = exec.Command(containerRunner, "network", "create", "runp-network")
	cmdLine = strings.Join(command.Args, " ")
	ui.Debugf("Creating network: %s", cmdLine)
	_, err = command.Output()
	if err != nil {
		return PreconditionVerifyResult{
			Vote:    Stop,
			Reasons: []string{fmt.Sprintf("Failed to read network creation command output: %s (%v)", cmdLine, err)},
		}
	}
	return PreconditionVerifyResult{
		Vote:    Proceed,
		Reasons: []string{},
	}
}
