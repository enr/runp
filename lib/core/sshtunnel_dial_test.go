package core

import (
	"net"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// silentServer accepts TCP connections and never speaks SSH.
func silentServer(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var conns []net.Conn
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			conns = append(conns, c)
		}
	}()
	t.Cleanup(func() { ln.Close() })
	return ln.Addr().String()
}

// The timeout must also bound the SSH handshake.
func TestDialSSH_HandshakeTimeout(t *testing.T) {
	addr := silentServer(t)
	config := &ssh.ClientConfig{
		User:            "test",
		Auth:            []ssh.AuthMethod{ssh.Password("test")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         300 * time.Millisecond,
	}
	done := make(chan error, 1)
	go func() {
		_, err := dialSSH(addr, config)
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("dialSSH succeeded against a server not speaking SSH")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("dialSSH did not honour the timeout during the handshake")
	}
}

// Stop must not wait for a connection that is still dialling the jump server.
func TestSSHTunnelCommandWrapper_StopDuringDial(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{Debug: false, Color: false})
	c := &SSHTunnelCommandWrapper{
		localAddress:  "127.0.0.1:0",
		jumpAddress:   silentServer(t),
		targetAddress: "127.0.0.1:1",
		config: &ssh.ClientConfig{
			User:            "test",
			Auth:            []ssh.AuthMethod{ssh.Password("test")},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         10 * time.Second,
		},
	}
	go c.Wait()
	var addr string
	deadline := time.Now().Add(2 * time.Second)
	for addr == "" {
		c.mu.Lock()
		if c.listener != nil {
			addr = c.listener.Addr().String()
		}
		c.mu.Unlock()
		if time.Now().After(deadline) {
			t.Fatal("tunnel not listening")
		}
		time.Sleep(10 * time.Millisecond)
	}
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	time.Sleep(200 * time.Millisecond) // let the tunnel start dialling

	stopped := make(chan struct{})
	go func() {
		c.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop blocked while a connection was dialling the jump server")
	}
}
