package core

import (
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"time"
)

// Shell represents the shell used to invoke command.
type Shell struct {
	Path string
	Args []string
}

// HostProcess implements RunpProcess.
type HostProcess struct {
	// command line
	CommandLine string `yaml:"command"`
	// executable
	Executable string
	Args       []string
	Shell      Shell
	// generic
	WorkingDir string `yaml:"workdir"`
	Env        map[string]string
	Await      AwaitCondition

	id            string
	cmd           *exec.Cmd
	vars          map[string]string
	preconditions []Precondition
	secretKey     string
	stopTimeout   string
}

// ID for the sub process
func (p *HostProcess) ID() string {
	return p.id
}

// SetID for the sub process
func (p *HostProcess) SetID(id string) {
	p.id = id
}

// SetPreconditions set preconditions.
func (p *HostProcess) SetPreconditions(preconditions []Precondition) {
	p.preconditions = preconditions
}

// VerifyPreconditions check if process can be started
func (p *HostProcess) VerifyPreconditions() error {
	var err error
	for _, p := range p.preconditions {
		err = p.Verify()
		if err != nil {
			return err
		}
	}
	return nil
}

// StopTimeout duration to wait to force kill process
func (p *HostProcess) StopTimeout() time.Duration {
	if p.stopTimeout != "" {
		d, err := time.ParseDuration(p.stopTimeout)
		if err != nil {
			return time.Duration(5) * time.Second
		}
		return d
	}
	return time.Duration(5) * time.Second
}

// StartCommand ho
func (p *HostProcess) StartCommand() (RunpCommand, error) {
	var cmd *exec.Cmd
	var err error
	if p.Executable != "" {
		cmd, err = p.buildCmdExecutable()
	} else {
		cmd, err = p.buildCmdCommandline()
	}
	if err != nil {
		return nil, err
	}
	cmd.Dir = p.WorkingDir
	p.cmd = cmd
	return &ExecCommandWrapper{
		cmd: cmd,
	}, nil
}

// StopCommand ho
func (p *HostProcess) StopCommand() RunpCommand {
	return &ExecCommandStopper{
		id:  p.id,
		cmd: p.cmd,
	}
}

// Dir for the sub process
func (p *HostProcess) Dir() string {
	return p.WorkingDir
}

// SetDir for the sub process
func (p *HostProcess) SetDir(wd string) {
	p.WorkingDir = wd
}

// String representation of process
func (p *HostProcess) String() string {
	return fmt.Sprintf("%T{id=%s}", p, p.ID())
}

func (p *HostProcess) buildCmdExecutable() (*exec.Cmd, error) {
	exe := p.Executable
	// var cmd *exec.Cmd
	p1, err := exec.LookPath(exe)
	if err != nil {
		ui.Debugf("Executable '%s' not found", exe)
		m := filepath.FromSlash(path.Join(p.WorkingDir, exe))
		p2, err := exec.LookPath(m)
		if err != nil {
			ui.WriteLinef("Executable for process '%s' not found. Tried '%s' and '%s' \n", p.ID(), exe, m)
			return nil, err
		}
		exe = p2
	} else {
		exe = p1
	}
	ui.Debugf("'exe' executable is in '%s'", exe)
	cliPreprocessor := newCliPreprocessor(p.vars)
	cmd := exec.Command(exe, cliPreprocessor.processArgs(p.Args)...)
	cmd.Env = p.resolveEnvironment()
	return cmd, nil
}

func (p *HostProcess) buildCmdCommandline() (*exec.Cmd, error) {
	// if commandline, use shell
	exe, args, err := p.resolveShell()
	if err != nil {
		return nil, err
	}
	ui.Debugf(`Process %s will be started using shell "%s" %q`, p.ID(), exe, args)
	cliPreprocessor := newCliPreprocessor(p.vars)
	cmd := exec.Command(exe, cliPreprocessor.processArgs(args)...)
	cmd.Env = p.resolveEnvironment()
	return cmd, nil
}

func (p *HostProcess) resolveShell() (string, []string, error) {
	shellCommand := p.CommandLine
	shell := defaultShell()
	if p.Shell.Path != "" {
		shell = p.Shell
	}
	return shell.Path, append(shell.Args, shellCommand), nil
}

func (p *HostProcess) resolveEnvironment() []string {
	environment := []string{}
	for _, item := range envAsArray(p.Env) {
		environment = append(environment, item)
	}
	return environment
}

// ShouldWait returns if the process has await set.
func (p *HostProcess) ShouldWait() bool {
	return (p.Await.Resource != "")
}

// AwaitResource returns the await resource.
func (p *HostProcess) AwaitResource() string {
	return p.Await.Resource
}

// AwaitTimeout returns the await timeout.
func (p *HostProcess) AwaitTimeout() string {
	return p.Await.Timeout
}

// IsStartable always true.
func (p *HostProcess) IsStartable() (bool, error) {
	return true, nil
}
