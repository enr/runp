package core

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// ExecCommandWrapper ...
type ExecCommandWrapper struct {
	// name string
	cmd *exec.Cmd
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
	return c.cmd.Start()
}

// Run ...
func (c *ExecCommandWrapper) Run() error {
	return c.cmd.Run()
}

// Stop ...
func (c *ExecCommandWrapper) Stop() error {
	return c.cmd.Process.Kill()
}

// Wait ...
func (c *ExecCommandWrapper) Wait() error {
	return c.cmd.Wait()
}

func (c *ExecCommandWrapper) String() string {
	return fmt.Sprintf("%T %s# %s", c, c.cmd.Dir, strings.Join(c.cmd.Args, " "))
}

// ExecCommandStopper is the component calling the actual command stopping the process.
type ExecCommandStopper struct {
	id  string
	cmd *exec.Cmd
}

// Pid ...
func (c *ExecCommandStopper) Pid() int {
	return c.cmd.Process.Pid
}

// Stdout ...
func (c *ExecCommandStopper) Stdout(stdout io.Writer) {
	c.cmd.Stdout = stdout
}

// Stderr ...
func (c *ExecCommandStopper) Stderr(stderr io.Writer) {
	c.cmd.Stderr = stderr
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
	p := c.cmd.Process
	if p == nil {
		ui.WriteLinef("Process %s not found, was actually started?", c.id)
		return nil
	}
	if c.cmd.ProcessState == nil || c.cmd.ProcessState.Exited() {
		return nil
	}
	return p.Kill()
}

// Wait ...
func (c *ExecCommandStopper) Wait() error {
	if c.cmd.ProcessState == nil || c.cmd.ProcessState.Exited() {
		return nil
	}
	return c.cmd.Wait()
}

func (c *ExecCommandStopper) String() string {
	return fmt.Sprintf("%T %s# %s", c, c.cmd.Dir, strings.Join(c.cmd.Args, " "))
}
