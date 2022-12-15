package core

import (
	"fmt"
	"io"
	"net"

	"golang.org/x/crypto/ssh"
)

// SSHTunnelCommandWrapper ...
type SSHTunnelCommandWrapper struct {
	config *ssh.ClientConfig

	localAddress  string
	jumpAddress   string
	targetAddress string

	// internal connection on local port
	localConnection net.Conn
	// ssh connection between localhost and ssh server
	localToJumpConnection *ssh.Client
	// connection between ssh server and target
	jumpToTargetConnection net.Conn

	stdout io.Writer
	stderr io.Writer
}

// Pid ...
func (c *SSHTunnelCommandWrapper) Pid() int {
	return -100
}

// Stdout ...
func (c *SSHTunnelCommandWrapper) Stdout(stdout io.Writer) {
	c.stdout = stdout
}

// Stderr ...
func (c *SSHTunnelCommandWrapper) Stderr(stderr io.Writer) {
	c.stderr = stderr
}

// Start ...
func (c *SSHTunnelCommandWrapper) Start() error {
	c.pf("Starting SSH tunnel %s -> %s -> %s", c.localAddress, c.jumpAddress, c.targetAddress)
	return nil
}

// Run ...
func (c *SSHTunnelCommandWrapper) Run() error {
	return nil
}

// Stop ...
func (c *SSHTunnelCommandWrapper) Stop() error {
	c.pf("Stopping SSH tunnel")
	var err error
	var errors multiError
	c.pf(`Closing SSH connection to target %v`, c.jumpToTargetConnection)
	if c.jumpToTargetConnection != nil {
		err := c.jumpToTargetConnection.Close()
		if err != nil {
			c.pf("Error closing connection to target: %s", err)
			errors = append(errors, err)
		}

	}
	c.pf(`Closing SSH connection to jump server %v`, c.localToJumpConnection)
	if c.localToJumpConnection != nil {
		err = c.localToJumpConnection.Close()
		if err != nil {
			c.pf("Error closing connection to jump server: %s", err)
			errors = append(errors, err)
		}
	}
	c.pf(`Closing local connection %v`, c.localConnection)
	if c.localConnection != nil {
		err = c.localConnection.Close()
		if err != nil {
			c.pf("Error closing local connection: %s", err)
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return errors
	}
	return nil
}

// Wait ...
func (c *SSHTunnelCommandWrapper) Wait() error {
	localListener, err := net.Listen("tcp", c.localAddress)
	if err != nil {
		c.pf("Error starting local listener %s %v", c.localAddress, err)
		return err
	}

	for {
		c.localConnection, err = localListener.Accept()
		if err != nil {
			c.pf("Error in local listener accept %v", err)
			return err
		}
		go func() {
			err = c.forward()
			ui.WriteLinef(`Error in forward: %+v`, err)
		}()
	}
}

func (c *SSHTunnelCommandWrapper) String() string {
	return fmt.Sprintf("%T (%d)", c, c.Pid())
}

func (c *SSHTunnelCommandWrapper) pf(format string, a ...interface{}) {
	if c.stdout == nil {
		return
	}
	fmt.Fprintf(c.stdout, format, a...)
}

func (c *SSHTunnelCommandWrapper) forward() error {
	var err error
	c.localToJumpConnection, err = ssh.Dial("tcp", c.jumpAddress, c.config)
	if err != nil {
		c.pf("Error connecting to jump server %s: %v", c.jumpAddress, err)
		return err
	}

	c.jumpToTargetConnection, err = c.localToJumpConnection.Dial("tcp", c.targetAddress)
	if err != nil {
		c.pf("Error connecting to target from jump server: %v", err)
		return err
	}

	// Copy localConnection.Reader to jumpToTargetConnection.Writer
	go func() {
		if c.localConnection == nil || c.jumpToTargetConnection == nil {
			c.pf("Missing connection: local=%v jump=%v", c.localConnection, c.jumpToTargetConnection)
			return
		}
		_, err = io.Copy(c.jumpToTargetConnection, c.localConnection)
		if err != nil {
			c.pf("\nError connecting local to jump server:\n%v\n", err)
		}
	}()

	// Copy jumpToTargetConnection.Reader to localConnection.Writer
	go func() {
		if c.localConnection == nil || c.jumpToTargetConnection == nil {
			c.pf("Missing connection: local=%v jump=%v", c.localConnection, c.jumpToTargetConnection)
			return
		}
		_, err = io.Copy(c.localConnection, c.jumpToTargetConnection)
		if err != nil {
			c.pf("\nError connecting jump server to local:\n%v\n", err)
		}
	}()
	return nil
}

// SSHTunnelCommandStopper is the component calling the actual command stopping the process.
type SSHTunnelCommandStopper struct {
	id  string
	cmd *SSHTunnelCommandWrapper
}

// Pid ...
func (c *SSHTunnelCommandStopper) Pid() int {
	return -300
}

// Stdout ...
func (c *SSHTunnelCommandStopper) Stdout(stdout io.Writer) {

}

// Stderr ...
func (c *SSHTunnelCommandStopper) Stderr(stderr io.Writer) {

}

// Start ...
func (c *SSHTunnelCommandStopper) Start() error {
	return c.cmd.Stop()
}

// Run ...
func (c *SSHTunnelCommandStopper) Run() error {
	return c.cmd.Stop()
}

// Stop ...
func (c *SSHTunnelCommandStopper) Stop() error {
	return c.cmd.Stop()
}

// Wait ...
func (c *SSHTunnelCommandStopper) Wait() error {
	return nil
}

func (c *SSHTunnelCommandStopper) String() string {
	return fmt.Sprintf("%T (%d)", c, c.Pid())
}
