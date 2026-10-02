package core

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"golang.org/x/crypto/ssh"
)

// SSHTunnelCommandWrapper forwards connections accepted on localAddress to
// targetAddress through an SSH connection to jumpAddress.
//
// Every accepted connection is handled with its own local and remote sockets;
// the SSH client is shared across connections and re-dialled when it breaks.
type SSHTunnelCommandWrapper struct {
	config *ssh.ClientConfig

	localAddress  string
	jumpAddress   string
	targetAddress string

	// mu guards every field below.
	mu       sync.Mutex
	listener net.Listener
	client   *ssh.Client
	conns    map[net.Conn]struct{}
	stopped  bool

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

// Stop closes the local listener, every open forwarded connection and the
// SSH client. It makes a running Wait return.
func (c *SSHTunnelCommandWrapper) Stop() error {
	c.pf("Stopping SSH tunnel")
	c.mu.Lock()
	c.stopped = true
	listener, client := c.listener, c.client
	c.listener, c.client = nil, nil
	conns := make([]net.Conn, 0, len(c.conns))
	for conn := range c.conns {
		conns = append(conns, conn)
	}
	c.conns = nil
	c.mu.Unlock()

	var errs multiError
	closeAll := func(what string, cl io.Closer) {
		if err := cl.Close(); err != nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, io.EOF) {
			c.pf("Error closing %s: %s", what, err)
			errs = append(errs, err)
		}
	}
	if listener != nil {
		closeAll("local listener", listener)
	}
	for _, conn := range conns {
		closeAll("forwarded connection", conn)
	}
	if client != nil {
		closeAll("SSH connection to jump server", client)
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

// Wait listens on the local address and forwards connections until Stop is called.
func (c *SSHTunnelCommandWrapper) Wait() error {
	listener, err := net.Listen("tcp", c.localAddress)
	if err != nil {
		c.pf("Failed to start local listener on %s: %v", c.localAddress, err)
		return err
	}
	c.mu.Lock()
	if c.stopped {
		c.mu.Unlock()
		listener.Close()
		return nil
	}
	c.listener = listener
	c.mu.Unlock()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if c.isStopped() {
				return nil
			}
			c.pf("Failed to accept connection on local listener: %v", err)
			return err
		}
		go c.handle(conn)
	}
}

func (c *SSHTunnelCommandWrapper) String() string {
	return fmt.Sprintf("%T (%d)", c, c.Pid())
}

func (c *SSHTunnelCommandWrapper) pf(format string, a ...interface{}) {
	if c.stdout == nil {
		return
	}
	fmt.Fprintf(c.stdout, format+"\n", a...)
}

func (c *SSHTunnelCommandWrapper) isStopped() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stopped
}

// track registers conn so Stop can close it. It returns false when the tunnel
// is already stopped; the caller must then close conn itself.
func (c *SSHTunnelCommandWrapper) track(conn net.Conn) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopped {
		return false
	}
	if c.conns == nil {
		c.conns = make(map[net.Conn]struct{})
	}
	c.conns[conn] = struct{}{}
	return true
}

func (c *SSHTunnelCommandWrapper) untrack(conn net.Conn) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.conns, conn)
}

// sshClient returns the shared SSH client, dialling the jump server if needed.
func (c *SSHTunnelCommandWrapper) sshClient() (*ssh.Client, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopped {
		return nil, errors.New("SSH tunnel stopped")
	}
	if c.client != nil {
		return c.client, nil
	}
	client, err := ssh.Dial("tcp", c.jumpAddress, c.config)
	if err != nil {
		return nil, err
	}
	c.client = client
	return client, nil
}

// dropClient closes client and forgets it, so the next connection re-dials.
func (c *SSHTunnelCommandWrapper) dropClient(client *ssh.Client) {
	c.mu.Lock()
	if c.client == client {
		c.client = nil
	}
	c.mu.Unlock()
	client.Close()
}

// handle forwards a single accepted local connection to the target.
func (c *SSHTunnelCommandWrapper) handle(local net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			ui.WriteLinef("Panic in SSH tunnel forward goroutine: %v", r)
			local.Close()
		}
	}()
	if !c.track(local) {
		local.Close()
		return
	}
	defer c.untrack(local)

	client, err := c.sshClient()
	if err != nil {
		c.pf("Failed to connect to jump server %s: %v", c.jumpAddress, err)
		local.Close()
		return
	}
	remote, err := client.Dial("tcp", c.targetAddress)
	if err != nil {
		c.pf("Failed to connect to target %s from jump server: %v", c.targetAddress, err)
		local.Close()
		// The client may be broken (e.g. jump server restarted): re-dial next time.
		c.dropClient(client)
		return
	}
	if !c.track(remote) {
		remote.Close()
		local.Close()
		return
	}
	defer c.untrack(remote)

	pipe(local, remote, c.pf)
}

// pipe copies data in both directions between a and b. When either direction
// ends, both connections are closed so the other copy returns as well.
func pipe(a, b net.Conn, pf func(string, ...interface{})) {
	var once sync.Once
	closeBoth := func() {
		once.Do(func() {
			a.Close()
			b.Close()
		})
	}
	var wg sync.WaitGroup
	wg.Add(2)
	cp := func(dst, src net.Conn, dir string) {
		defer wg.Done()
		defer closeBoth()
		if _, err := io.Copy(dst, src); err != nil && !errors.Is(err, net.ErrClosed) {
			pf("Error copying data %s: %v", dir, err)
		}
	}
	go cp(b, a, "from local to target")
	go cp(a, b, "from target to local")
	wg.Wait()
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
