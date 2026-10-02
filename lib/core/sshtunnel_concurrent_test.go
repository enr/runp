//go:build darwin || freebsd || linux || netbsd || openbsd

package core

import (
	"fmt"
	"io"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// Concurrent clients of the same tunnel must each get their own stream:
// data from one client must never be delivered to another.
func TestSSHTunnelConcurrentConnectionsAreIsolated(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{Debug: false, Color: false})

	sshKey, err := filepath.Abs("../../testdata/keys/runp")
	if err != nil {
		t.Fatal(err)
	}
	local := Endpoint{Host: "127.0.0.1", Port: 3097}
	jump := Endpoint{Host: "localhost", Port: 6667}

	jumpListener, _, _ := testListener(jump, t)
	defer jumpListener.Close()
	sshConfig, err := sshServerConfig("test", "test", sshKey)
	if err != nil {
		t.Fatal(err)
	}
	startSSHServer(jumpListener, sshConfig, t)

	// Target: TCP echo server.
	echo, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer echo.Close()
	go func() {
		for {
			conn, err := echo.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				io.Copy(conn, conn)
			}()
		}
	}()
	target := Endpoint{Host: "127.0.0.1", Port: echo.Addr().(*net.TCPAddr).Port}

	tunnel := &SSHTunnelProcess{
		User:   "test",
		Auth:   Auth{Secret: "test"},
		Local:  local,
		Jump:   jump,
		Target: target,
		KnownHostsFile: knownHostsFileForTest(t,
			fmt.Sprintf("[%s]:%d", jump.Host, jump.Port), "../../testdata/keys/runp.pub"),
	}
	cmd, err := tunnel.StartCommand()
	if err != nil {
		t.Fatal(err)
	}
	go cmd.Wait()
	defer cmd.Stop()

	var conns []net.Conn
	deadline := time.Now().Add(2 * time.Second)
	for len(conns) == 0 {
		c, err := net.Dial("tcp", local.String())
		if err == nil {
			conns = append(conns, c)
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("tunnel not listening: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	const clients = 8
	for len(conns) < clients {
		c, err := net.Dial("tcp", local.String())
		if err != nil {
			t.Fatal(err)
		}
		conns = append(conns, c)
	}

	var wg sync.WaitGroup
	for i, c := range conns {
		wg.Add(1)
		go func(i int, c net.Conn) {
			defer wg.Done()
			defer c.Close()
			c.SetDeadline(time.Now().Add(5 * time.Second))
			for round := 0; round < 20; round++ {
				msg := fmt.Sprintf("client-%02d-round-%02d", i, round)
				if _, err := c.Write([]byte(msg)); err != nil {
					t.Errorf("client %d write: %v", i, err)
					return
				}
				buf := make([]byte, len(msg))
				if _, err := io.ReadFull(c, buf); err != nil {
					t.Errorf("client %d read: %v", i, err)
					return
				}
				if string(buf) != msg {
					t.Errorf("client %d got %q, want %q", i, buf, msg)
					return
				}
			}
		}(i, c)
	}
	wg.Wait()
}
