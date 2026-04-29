//go:build darwin || freebsd || linux || netbsd || openbsd

package core

import (
	"fmt"
	"os"
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
