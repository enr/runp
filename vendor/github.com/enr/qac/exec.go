package qac

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/enr/runcmd"
)

type executionResult struct {
	success   bool
	exitCode  int
	stdout    string
	stderr    string
	execution string
	err       error
	timedOut  bool
	duration  time.Duration
}

// The actual command executor used from launcher.
type executor interface {
	execute(c Command) executionResult
}

type runcmdExecutor struct {
}

func (e *runcmdExecutor) execute(c Command) executionResult {
	var timeout time.Duration
	if c.Timeout != "" {
		if d, err := time.ParseDuration(c.Timeout); err == nil && d > 0 {
			timeout = d
		}
	}
	if timeout > 0 || c.Stdin != "" || c.StdinFile != "" {
		return e.executeDirect(c, timeout)
	}
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

// executeDirect runs a command via exec.Cmd, supporting optional timeout and stdin.
// timeout == 0 means no timeout.
func (e *runcmdExecutor) executeDirect(c Command, timeout time.Duration) executionResult {
	var ctx context.Context
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	fullCmd, cmd, err := e.buildExecCmd(ctx, c)
	if err != nil {
		return executionResult{exitCode: -1, err: err}
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if c.WorkingDir != "" {
		cmd.Dir = c.WorkingDir
	}
	if len(c.Env) > 0 {
		cmd.Env = mergeEnv(c.Env)
	}

	if c.Stdin != "" {
		cmd.Stdin = strings.NewReader(c.Stdin)
	} else if c.StdinFile != "" {
		f, openErr := os.Open(c.StdinFile)
		if openErr != nil {
			return executionResult{exitCode: -1, err: fmt.Errorf("opening stdin_file %q: %w", c.StdinFile, openErr)}
		}
		defer f.Close()
		cmd.Stdin = f
	} else {
		cmd.Stdin = io.Reader(nil)
	}

	runErr := cmd.Run()
	timedOut := errors.Is(ctx.Err(), context.DeadlineExceeded)

	exitCode := 0
	if runErr != nil {
		if ee, ok := runErr.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			exitCode = -1
		}
	}
	if timedOut {
		exitCode = -1
	}

	return executionResult{
		success:   runErr == nil,
		exitCode:  exitCode,
		stdout:    stdout.String(),
		stderr:    stderr.String(),
		err:       runErr,
		execution: fullCmd,
		timedOut:  timedOut,
	}
}

// buildExecCmd constructs an exec.Cmd with context.
// When c.Exe is set the process is started directly (no shell).
// When only c.Cli is set it is executed via $SHELL -c (or cmd /C on Windows).
func (e *runcmdExecutor) buildExecCmd(ctx context.Context, c Command) (string, *exec.Cmd, error) {
	exe := c.Exe
	args := c.Args

	if c.Extension.isSet() {
		exe = fmt.Sprintf(`%s%s`, exe, c.Extension.get())
	}

	if exe == "" && c.Cli == "" {
		return "", nil, fmt.Errorf("no executable or command line specified")
	}

	var fullCmd string
	if exe == "" {
		shell := os.Getenv("SHELL")
		if shell == "" {
			if runtime.GOOS == "windows" {
				shell = "cmd"
			} else {
				shell = "/bin/sh"
			}
		}
		if runtime.GOOS == "windows" {
			args = []string{"/C", c.Cli}
		} else {
			args = []string{"-c", c.Cli}
		}
		exe = shell
		fullCmd = c.Cli
	} else {
		fullCmd = strings.TrimSpace(exe + " " + strings.Join(args, " "))
	}

	return fullCmd, exec.CommandContext(ctx, exe, args...), nil
}

// mergeEnv overlays custom key=value pairs on top of the inherited environment.
func mergeEnv(custom map[string]string) []string {
	env := os.Environ()
	for k, v := range custom {
		env = append(env, k+"="+v)
	}
	return env
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
