package core

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// ExecCommandWrapper is wrapper for *exec.Cmd
type ExecCommandWrapper struct {
	// name string
	cmd *exec.Cmd

	// mu guards process and exited, which are written by the goroutines
	// calling Start and Wait and read by the shutdown goroutine.
	mu      sync.Mutex
	process *os.Process
	exited  bool
}

// Pid return PID for this command wrapper
func (c *ExecCommandWrapper) Pid() int {
	return c.cmd.Process.Pid
}

// Stdout set the stdout writer
func (c *ExecCommandWrapper) Stdout(stdout io.Writer) {
	c.cmd.Stdout = stdout
}

// Stderr  set the stderr writer
func (c *ExecCommandWrapper) Stderr(stderr io.Writer) {
	c.cmd.Stderr = stderr
}

// Start ...
func (c *ExecCommandWrapper) Start() error {
	if err := c.cmd.Start(); err != nil {
		return err
	}
	c.mu.Lock()
	c.process = c.cmd.Process
	c.mu.Unlock()
	return nil
}

// Run ...
func (c *ExecCommandWrapper) Run() error {
	if err := c.Start(); err != nil {
		return err
	}
	return c.Wait()
}

// Stop ...
func (c *ExecCommandWrapper) Stop() error {
	return c.stopWithGracefulShutdown(5*time.Second, "")
}

// stopWithGracefulShutdown implements graceful shutdown (platform-specific implementation)
func (c *ExecCommandWrapper) stopWithGracefulShutdown(timeout time.Duration, id string) error {
	p := c.startedProcess()
	if p == nil {
		if id != "" {
			ui.WriteLinef("Process %s not found: process may not have been started", id)
		}
		return nil
	}
	return stopProcess(p, c.hasExited, timeout, id)
}

// Wait waits for the command to exit.
func (c *ExecCommandWrapper) Wait() error {
	err := c.cmd.Wait()
	c.mu.Lock()
	c.exited = true
	c.mu.Unlock()
	return err
}

func (c *ExecCommandWrapper) startedProcess() *os.Process {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.process
}

func (c *ExecCommandWrapper) hasExited() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.exited
}

func (c *ExecCommandWrapper) String() string {
	return fmt.Sprintf("%T %s# %s", c, c.cmd.Dir, strings.Join(c.cmd.Args, " "))
}

// ExecCommandStopper is the component calling the actual command stopping the process.
type ExecCommandStopper struct {
	id string
	// wrapper is the running command; when set it is the only source of
	// process state, so no exec.Cmd field is read concurrently with Wait.
	wrapper *ExecCommandWrapper
	// cmd is used when no wrapper is available.
	cmd     *exec.Cmd
	timeout time.Duration
}

func (c *ExecCommandStopper) execCmd() *exec.Cmd {
	if c.wrapper != nil {
		return c.wrapper.cmd
	}
	return c.cmd
}

// Pid ...
func (c *ExecCommandStopper) Pid() int {
	if c.wrapper != nil {
		if p := c.wrapper.startedProcess(); p != nil {
			return p.Pid
		}
		return 0
	}
	return c.cmd.Process.Pid
}

// Stdout ...
func (c *ExecCommandStopper) Stdout(stdout io.Writer) {
	if cmd := c.execCmd(); cmd != nil {
		cmd.Stdout = stdout
	}
}

// Stderr ...
func (c *ExecCommandStopper) Stderr(stderr io.Writer) {
	if cmd := c.execCmd(); cmd != nil {
		cmd.Stderr = stderr
	}
}

// Start ...
func (c *ExecCommandStopper) Start() error {
	return c.Stop()
}

// Run ...
func (c *ExecCommandStopper) Run() error {
	return c.Stop()
}

// Stop ...
func (c *ExecCommandStopper) Stop() error {
	return c.stopWithGracefulShutdown(c.timeout)
}

// stopWithGracefulShutdown implements graceful shutdown (platform-specific implementation)
func (c *ExecCommandStopper) stopWithGracefulShutdown(timeout time.Duration) error {
	if c.wrapper != nil {
		return c.wrapper.stopWithGracefulShutdown(timeout, c.id)
	}
	if c.cmd == nil || c.cmd.Process == nil {
		ui.WriteLinef("Process %s not found: process may not have been started", c.id)
		return nil
	}
	cmd := c.cmd
	exited := func() bool { return cmd.ProcessState != nil && cmd.ProcessState.Exited() }
	return stopProcess(cmd.Process, exited, timeout, c.id)
}

// Wait is a no-op: the executor owns the single Wait() call on the process.
func (c *ExecCommandStopper) Wait() error {
	return nil
}

func (c *ExecCommandStopper) String() string {
	cmd := c.execCmd()
	if cmd == nil {
		return fmt.Sprintf("%T (not started)", c)
	}
	return fmt.Sprintf("%T %s# %s", c, cmd.Dir, strings.Join(cmd.Args, " "))
}
