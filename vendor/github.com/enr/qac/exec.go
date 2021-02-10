package qac

import (
	"fmt"

	"github.com/enr/runcmd"
)

type executionResult struct {
	success   bool
	exitCode  int
	stdout    string
	stderr    string
	execution string
	err       error
}

// The actual command executor used from launcher.
type executor interface {
	execute(c Command) executionResult
}

type runcmdExecutor struct {
}

func (e *runcmdExecutor) execute(c Command) executionResult {
	command := e.toRuncmd(c)
	res := command.Run()
	return executionResult{
		success:   res.Success(),
		exitCode:  res.ExitStatus(),
		stdout:    res.Stdout().String(),
		stderr:    res.Stderr().String(),
		err:       res.Error(),
		execution: command.FullCommand(),
	}
}

func (e *runcmdExecutor) toRuncmd(command Command) *runcmd.Command {
	exe := command.Exe
	if command.Extension.isSet() {
		exe = fmt.Sprintf(`%s%s`, command.Exe, command.Extension.get())
	}
	c := &runcmd.Command{
		Exe:         exe,
		Args:        command.Args,
		CommandLine: command.Cli,
		WorkingDir:  command.WorkingDir,
		Env:         command.Env,
	}
	return c
}
