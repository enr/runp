//go:build darwin || freebsd || linux || netbsd || openbsd

package core

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

// pubKeyKnownHostsLine parses an OpenSSH public key file and returns the
// "key-type base64" portion suitable for embedding in a known_hosts entry.
func pubKeyKnownHostsLine(t *testing.T, pubKeyPath string) string {
	t.Helper()
	abs, err := filepath.Abs(pubKeyPath)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	pub, _, _, _, err := ssh.ParseAuthorizedKey(data)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pub)))
}

// writeKnownHostsFile creates a temp known_hosts file with a single entry.
// hostPattern is e.g. "[localhost]:6666", pubKeyLine is "key-type base64".
func writeKnownHostsFile(t *testing.T, hostPattern, pubKeyLine string) string {
	t.Helper()
	f, err := os.CreateTemp("", "test_known_hosts_*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(f.Name()) })
	fmt.Fprintf(f, "%s %s\n", hostPattern, pubKeyLine)
	f.Close()
	return f.Name()
}

// TestSSHTunnelProcess_HostKeyVerification_RejectsWrongKey verifies that
// ssh.Dial is rejected when the server presents a key not in known_hosts.
// This test exposes the MITM vulnerability: it FAILs before the fix
// (InsecureIgnoreHostKey allows any key) and PASSes after it.
func TestSSHTunnelProcess_HostKeyVerification_RejectsWrongKey(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{Debug: true, Color: false})

	sshUser := "test"
	sshSecret := "test"

	sshKey, err := filepath.Abs("../../testdata/keys/runp")
	if err != nil {
		t.Fatal(err)
	}

	listener, host, port := testListener(Endpoint{Port: 0}, t)
	defer listener.Close()

	sshConfig, err := sshServerConfig(sshUser, sshSecret, sshKey)
	if err != nil {
		t.Fatal(err)
	}
	startSSHServer(listener, sshConfig, t)

	// known_hosts contains a DIFFERENT key — not what the server presents.
	wrongKeyLine := pubKeyKnownHostsLine(t, "../../testdata/ssh/keys/ssh_host_rsa_key.pub")
	hostPattern := fmt.Sprintf("[%s]:%d", host, port)
	knownHostsFile := writeKnownHostsFile(t, hostPattern, wrongKeyLine)

	p := &SSHTunnelProcess{
		User:           sshUser,
		Auth:           Auth{Secret: sshSecret},
		Jump:           Endpoint{Host: host, Port: port},
		KnownHostsFile: knownHostsFile,
		vars:           map[string]string{},
	}

	clientConfig, err := p.resolveSSHCommandConfiguration()
	if err != nil {
		t.Fatalf("resolveSSHCommandConfiguration failed: %v", err)
	}

	_, err = ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), clientConfig)
	if err == nil {
		t.Error("expected ssh.Dial to fail due to host key mismatch — SSH host key verification is not enforced (MITM vulnerability)")
	}
}

// startExecSSHServer starts a minimal SSH server that handles exec requests,
// replies immediately (before waiting for the process), and sends the exit-status
// so session.Run on the client side receives the correct exit code.
// This avoids the deadlock in easyssh.SessionHandler where req.Reply is sent
// only after command.Wait(), causing the client's SendRequest to block forever.
func startExecSSHServer(t *testing.T, listener net.Listener, config *ssh.ServerConfig) {
	t.Helper()
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			_, chans, reqs, err := ssh.NewServerConn(conn, config)
			if err != nil {
				continue
			}
			go ssh.DiscardRequests(reqs)
			go func() {
				for newChan := range chans {
					if newChan.ChannelType() != "session" {
						newChan.Reject(ssh.UnknownChannelType, "unknown channel type")
						continue
					}
					ch, requests, err := newChan.Accept()
					if err != nil {
						return
					}
					go serveExecSession(ch, requests)
				}
			}()
		}
	}()
}

func serveExecSession(ch ssh.Channel, reqs <-chan *ssh.Request) {
	defer ch.Close()
	for req := range reqs {
		switch req.Type {
		case "exec":
			var payload struct{ Command string }
			if err := ssh.Unmarshal(req.Payload, &payload); err != nil {
				req.Reply(false, nil)
				return
			}
			req.Reply(true, nil)
			cmd := exec.Command("sh", "-c", payload.Command)
			cmd.Stdout = ch
			cmd.Stderr = ch.Stderr()
			exitCode := uint32(0)
			if err := cmd.Run(); err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = uint32(exitErr.ExitCode())
				} else {
					exitCode = 1
				}
			}
			ch.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{exitCode}))
			return
		default:
			if req.WantReply {
				req.Reply(false, nil)
			}
		}
	}
}

// TestExecuteCmd_FailingCommandReturnsError verifies that executeCmd propagates
// a non-zero exit status from the remote command as an error instead of
// silently succeeding.  Before the fix, session.Run() errors were discarded
// and the function always returned nil — making VerifyPreconditions useless.
func TestExecuteCmd_FailingCommandReturnsError(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{Debug: true, Color: false})

	sshUser := "test"
	sshSecret := "test"

	sshKey, err := filepath.Abs("../../testdata/keys/runp")
	if err != nil {
		t.Fatal(err)
	}

	listener, host, port := testListener(Endpoint{Port: 0}, t)
	defer listener.Close()

	sshConfig, err := sshServerConfig(sshUser, sshSecret, sshKey)
	if err != nil {
		t.Fatal(err)
	}
	startExecSSHServer(t, listener, sshConfig)

	correctKeyLine := pubKeyKnownHostsLine(t, "../../testdata/keys/runp.pub")
	hostPattern := fmt.Sprintf("[%s]:%d", host, port)
	knownHostsFile := writeKnownHostsFile(t, hostPattern, correctKeyLine)

	p := &SSHTunnelProcess{
		User:           sshUser,
		Auth:           Auth{Secret: sshSecret},
		Jump:           Endpoint{Host: host, Port: port},
		KnownHostsFile: knownHostsFile,
		vars:           map[string]string{},
	}

	_, err = p.executeCmd("false")
	if err == nil {
		t.Error("executeCmd should return an error when the remote command exits with non-zero status — session.Run() error is being silently discarded")
	}
}

// TestSSHTunnelProcess_HostKeyVerification_AcceptsCorrectKey verifies that
// ssh.Dial succeeds when the server presents the key in known_hosts.
func TestSSHTunnelProcess_HostKeyVerification_AcceptsCorrectKey(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{Debug: true, Color: false})

	sshUser := "test"
	sshSecret := "test"

	sshKey, err := filepath.Abs("../../testdata/keys/runp")
	if err != nil {
		t.Fatal(err)
	}

	listener, host, port := testListener(Endpoint{Port: 0}, t)
	defer listener.Close()

	sshConfig, err := sshServerConfig(sshUser, sshSecret, sshKey)
	if err != nil {
		t.Fatal(err)
	}
	startSSHServer(listener, sshConfig, t)

	// known_hosts contains the CORRECT key — the one the server presents.
	correctKeyLine := pubKeyKnownHostsLine(t, "../../testdata/keys/runp.pub")
	hostPattern := fmt.Sprintf("[%s]:%d", host, port)
	knownHostsFile := writeKnownHostsFile(t, hostPattern, correctKeyLine)

	p := &SSHTunnelProcess{
		User:           sshUser,
		Auth:           Auth{Secret: sshSecret},
		Jump:           Endpoint{Host: host, Port: port},
		KnownHostsFile: knownHostsFile,
		vars:           map[string]string{},
	}

	clientConfig, err := p.resolveSSHCommandConfiguration()
	if err != nil {
		t.Fatalf("resolveSSHCommandConfiguration failed: %v", err)
	}

	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), clientConfig)
	if err != nil {
		t.Errorf("unexpected error with correct host key in known_hosts: %v", err)
		return
	}
	conn.Close()
}
