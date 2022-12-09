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

	id            string
	vars          map[string]string
	preconditions []Precondition
	secretKey     string
	stopTimeout   string
}

// ID for the sub process
func (p *ContainerProcess) ID() string {
	return p.id
}

// SetID for the sub process
func (p *ContainerProcess) SetID(id string) {
	p.id = id
}

// StartCommand ho
func (p *ContainerProcess) StartCommand() (RunpCommand, error) {
	cmd, err := p.buildCmdImage()
	if err != nil {
		return nil, err
	}
	return &ExecCommandWrapper{
		cmd: cmd,
	}, nil
}

// StopCommand ho
func (p *ContainerProcess) StopCommand() RunpCommand {
	cl := fmt.Sprintf(`docker stop %s`, p.buildContainerName())
	cmd, err := cmd(cl)
	if err != nil {
		// TBD...
		ui.WriteLinef("Error building command line '%s': %v", cl, err)
	}
	return &ExecCommandWrapper{
		cmd: cmd,
	}
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

	cliPreprocessor := newCliPreprocessor(p.vars)
	var sb strings.Builder
	// rm Automatically remove the container when it exits
	sb.WriteString("docker run -t ")
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
	for name, val := range p.Env {
		sb.WriteString(`-e "`)
		sb.WriteString(name)
		sb.WriteString("=")
		sb.WriteString(os.ExpandEnv(val))
		sb.WriteString(`" `)
	}
	sb.WriteString(img)
	// command
	// command: ["-Djboss.http.port=8080", "-Djboss.management.http.port=9990"]
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
	return (p.Await.Resource != "")
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

// IsStartable always true.
func (p *ContainerProcess) IsStartable() (bool, error) {
	docker, err := exec.LookPath("docker")
	if err != nil {
		ui.WriteLinef("Unable to find docker executable: %v", err)
		return false, err
	}
	cn := p.buildContainerName()
	cmdLine := fmt.Sprintf("%s ps -aq -f name=%s", docker, cn)
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
	ui.Debugf("IsStartable command output:\n%s", so)
	if so != "" {
		ui.WriteLinef("Container %s not startable as it is already running: %s", cn, so)
		return false, nil
	}
	return true, nil
}

// SetPreconditions set preconditions.
func (p *ContainerProcess) SetPreconditions(preconditions []Precondition) {
	p.preconditions = preconditions
}

// VerifyPreconditions check if process can be started
func (p *ContainerProcess) VerifyPreconditions() error {
	var err error
	for _, p := range p.preconditions {
		err = p.Verify()
		if err != nil {
			return err
		}
	}

	docker, err := exec.LookPath("docker")
	if err != nil {
		ui.WriteLinef("Unable to find docker executable: %v", err)
		return err
	}
	cmdLine := fmt.Sprintf("%s network ls -q --filter name=runp-network --format '{{ .Name }}'", docker)
	ui.Debugf("PRECONDITIONS command:\n%s", cmdLine)
	command, err := cmd(cmdLine)
	if err != nil {
		return err
	}
	out, err := command.Output()
	if err != nil {
		return err
	}
	so := strings.TrimSpace(string(out))
	ui.Debugf("PRECONDITIONS command output:\n<%s>", so)
	if so == "runp-network" {
		return nil
	}
	cmdLine = fmt.Sprintf("%s network create runp-network", docker)
	ui.Debugf("PRECONDITIONS command:\n%s", cmdLine)
	command, err = cmd(cmdLine)
	if err != nil {
		return err
	}
	out, err = command.Output()
	if err != nil {
		return err
	}
	return nil
}
