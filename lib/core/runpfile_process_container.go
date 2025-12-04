package core

import (
	"fmt"
	"os"
	"os/exec"
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

// StopCommand returns the command stopping the process.
func (p *ContainerProcess) StopCommand() (RunpCommand, error) {
	containerRunner, err := exec.LookPath(p.environmentSettings.ContainerRunnerExe)
	if err != nil {
		ui.WriteLinef("Container runner executable not found: %s (%v)", p.environmentSettings.ContainerRunnerExe, err)
		return nil, err
	}
	cl := fmt.Sprintf(`%s stop %s`, containerRunner, p.buildContainerName())
	cmd, err := cmd(cl)
	if err != nil {
		ui.WriteLinef("Failed to build stop command: %s (%v)", cl, err)
		return nil, err
	}
	return &ExecCommandWrapper{
		cmd: cmd,
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

func (p *ContainerProcess) buildCmdLine() string {
	img := p.Image
	ui.Debugf("Run image '%s'\n", img)

	containerRunner, err := exec.LookPath(p.environmentSettings.ContainerRunnerExe)
	if err != nil {
		ui.WriteLinef("Container runner executable not found: %s (%v)", p.environmentSettings.ContainerRunnerExe, err)
		return ""
	}
	cliPreprocessor := newCliPreprocessor(p.vars)
	var sb strings.Builder
	// rm Automatically remove the container when it exits
	sb.WriteString(containerRunner)
	sb.WriteString(" run -t ")
	if !p.SkipRm {
		sb.WriteString("--rm ")
	}
	sb.WriteString("--name ")
	sb.WriteString(p.buildContainerName())
	sb.WriteString(" ")

	// network: --network it-network
	sb.WriteString("--network runp-network ")
	// --label , -l 		Set meta data on a container
	// --link
	// --shm-size
	if p.ShmSize != "" {
		sb.WriteString("--shm-size ")
		sb.WriteString(p.ShmSize)
		sb.WriteString(" ")
	}
	// --user
	// --volume
	for _, volume := range p.Volumes {
		sb.WriteString("--volume ")
		sb.WriteString(cliPreprocessor.process(volume))
		sb.WriteString(" ")
	}
	// --volumes-from
	for _, volume := range p.VolumesFrom {
		sb.WriteString("--volumes-from ")
		sb.WriteString(containerNamePrefix)
		sb.WriteString(cliPreprocessor.process(volume))
		sb.WriteString(" ")
	}
	// --mount
	for _, m := range p.Mounts {
		sb.WriteString("--mount ")
		sb.WriteString(cliPreprocessor.process(m))
		sb.WriteString(" ")
	}
	// --workdir
	if p.WorkingDir != "" {
		sb.WriteString("--workdir ")
		sb.WriteString(p.WorkingDir)
		sb.WriteString(" ")
	}

	for _, ports := range p.Ports {
		sb.WriteString("-p ")
		sb.WriteString(ports)
		sb.WriteString(" ")
	}
	// Process env with current vars
	for name, val := range p.Env {
		sb.WriteString(`-e "`)
		sb.WriteString(name)
		sb.WriteString("=")
		processedVal := cliPreprocessor.process(val)
		sb.WriteString(os.ExpandEnv(processedVal))
		sb.WriteString(`" `)
	}
	sb.WriteString(img)
	// command
	if p.Command != "" {
		sb.WriteString(` `)
		sb.WriteString(p.Command)
		sb.WriteString(` `)
	}
	return sb.String()
}

func (p *ContainerProcess) buildCmdImage() (*exec.Cmd, error) {
	cl := p.buildCmdLine()
	cliPreprocessor := newCliPreprocessor(p.vars)
	cl = cliPreprocessor.process(cl)
	ui.Debugf("Container command:\n%s", cl)
	return cmd(cl)
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
	containerRunner, err := exec.LookPath(p.environmentSettings.ContainerRunnerExe)
	if err != nil {
		ui.WriteLinef("Unable to find container runner %s executable: %v", p.environmentSettings.ContainerRunnerExe, err)
		return false, err
	}
	cn := p.buildContainerName()
	cmdLine := fmt.Sprintf("%s ps -aq -f name=%s", containerRunner, cn)
	ui.Debugf("IsStartable command:\n%s", cmdLine)
	cmd, err := cmd(cmdLine)
	if err != nil {
		return false, err
	}
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
	var err error
	containerRunner, err := exec.LookPath(p.environmentSettings.ContainerRunnerExe)
	if err != nil {
		return PreconditionVerifyResult{
			Vote:    Stop,
			Reasons: []string{fmt.Sprintf("Container runner executable not found: %s (%v)", p.environmentSettings.ContainerRunnerExe, err)},
		}
	}
	cmdLine := fmt.Sprintf("%s network ls -q --filter name=runp-network --format '{{ .Name }}'", containerRunner)
	ui.Debugf("Checking network precondition: %s", cmdLine)
	command, err := cmd(cmdLine)
	if err != nil {
		return PreconditionVerifyResult{
			Vote:    Stop,
			Reasons: []string{fmt.Sprintf("Failed to execute network check command: %s (%v)", cmdLine, err)},
		}
	}
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
	cmdLine = fmt.Sprintf("%s network create runp-network", containerRunner)
	ui.Debugf("Creating network: %s", cmdLine)
	command, err = cmd(cmdLine)
	if err != nil {
		return PreconditionVerifyResult{
			Vote:    Stop,
			Reasons: []string{fmt.Sprintf("Failed to create network: %s (%v)", cmdLine, err)},
		}
	}
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
