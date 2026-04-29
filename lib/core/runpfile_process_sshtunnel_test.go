package core

import (
	"os"
	"testing"
	"time"
)

func TestSSHTunnelProcess_SetID(t *testing.T) {
	p := &SSHTunnelProcess{}
	p.SetID("test-id")
	if p.ID() != "test-id" {
		t.Errorf("Expected ID 'test-id', got '%s'", p.ID())
	}
}

func TestSSHTunnelProcess_Dir(t *testing.T) {
	p := &SSHTunnelProcess{
		WorkingDir: "/test/dir",
	}
	if p.Dir() != "/test/dir" {
		t.Errorf("Expected Dir '/test/dir', got '%s'", p.Dir())
	}
}

func TestSSHTunnelProcess_SetDir(t *testing.T) {
	p := &SSHTunnelProcess{}
	p.SetDir("/new/dir")
	if p.Dir() != "/new/dir" {
		t.Errorf("Expected Dir '/new/dir', got '%s'", p.Dir())
	}
}

func TestSSHTunnelProcess_String(t *testing.T) {
	p := &SSHTunnelProcess{
		id: "test-process",
	}
	str := p.String()
	if str == "" {
		t.Error("String() should return non-empty string")
	}
	expected := "*core.SSHTunnelProcess{id=test-process}"
	if str != expected {
		t.Errorf("Expected String '%s', got '%s'", expected, str)
	}
}

func TestSSHTunnelProcess_ShouldWait(t *testing.T) {
	tests := []struct {
		name     string
		await    AwaitCondition
		expected bool
	}{
		{
			name:     "with timeout",
			await:    AwaitCondition{Timeout: "5s"},
			expected: true,
		},
		{
			name:     "without timeout",
			await:    AwaitCondition{Timeout: ""},
			expected: false,
		},
		{
			name:     "empty await",
			await:    AwaitCondition{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &SSHTunnelProcess{
				Await: tt.await,
			}
			if p.ShouldWait() != tt.expected {
				t.Errorf("Expected ShouldWait() %v, got %v", tt.expected, p.ShouldWait())
			}
		})
	}
}

func TestSSHTunnelProcess_AwaitResource(t *testing.T) {
	p := &SSHTunnelProcess{
		Await: AwaitCondition{
			Resource: "http://localhost:8080",
		},
	}
	if p.AwaitResource() != "http://localhost:8080" {
		t.Errorf("Expected AwaitResource 'http://localhost:8080', got '%s'", p.AwaitResource())
	}
}

func TestSSHTunnelProcess_AwaitTimeout(t *testing.T) {
	p := &SSHTunnelProcess{
		Await: AwaitCondition{
			Timeout: "10s",
		},
	}
	if p.AwaitTimeout() != "10s" {
		t.Errorf("Expected AwaitTimeout '10s', got '%s'", p.AwaitTimeout())
	}
}

func TestSSHTunnelProcess_IsStartable(t *testing.T) {
	p := &SSHTunnelProcess{}
	startable, err := p.IsStartable()
	if err != nil {
		t.Errorf("IsStartable() should not return error, got %v", err)
	}
	if !startable {
		t.Error("IsStartable() should return true")
	}
}

func TestSSHTunnelProcess_StopTimeout(t *testing.T) {
	tests := []struct {
		name        string
		stopTimeout string
		expected    time.Duration
	}{
		{
			name:        "valid duration",
			stopTimeout: "10s",
			expected:    10 * time.Second,
		},
		{
			name:        "invalid duration",
			stopTimeout: "invalid",
			expected:    5 * time.Second,
		},
		{
			name:        "empty timeout",
			stopTimeout: "",
			expected:    5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &SSHTunnelProcess{
				stopTimeout: tt.stopTimeout,
			}
			timeout := p.StopTimeout()
			if timeout != tt.expected {
				t.Errorf("Expected StopTimeout %v, got %v", tt.expected, timeout)
			}
		})
	}
}

func TestSSHTunnelProcess_StopCommand(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *SSHTunnelProcess
		wantErr bool
	}{
		{
			name: "with command set",
			setup: func() *SSHTunnelProcess {
				p := &SSHTunnelProcess{}
				cmd := &SSHTunnelCommandWrapper{}
				p.cmd = cmd
				return p
			},
			wantErr: false,
		},
		{
			name: "without command set",
			setup: func() *SSHTunnelProcess {
				return &SSHTunnelProcess{}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.setup()
			cmd, err := p.StopCommand()
			if (err != nil) != tt.wantErr {
				t.Errorf("StopCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && cmd == nil {
				t.Error("StopCommand() should return non-nil command when no error")
			}
		})
	}
}

func TestSSHTunnelProcess_resolveEnvironment(t *testing.T) {
	p := &SSHTunnelProcess{
		Env: map[string]string{
			"VAR1": "value1",
			"VAR2": "value2",
		},
		vars: map[string]string{
			"test": "replaced",
		},
	}
	env := p.resolveEnvironment()
	if len(env) == 0 {
		t.Error("resolveEnvironment() should return non-empty environment")
	}
	// Verifica che le variabili siano presenti
	foundVar1 := false
	for _, e := range env {
		if e == "VAR1=value1" {
			foundVar1 = true
			break
		}
	}
	if !foundVar1 {
		t.Errorf("Expected VAR1=value1 in environment, got %v", env)
	}
}

func TestSSHTunnelProcess_resolveEnvironment_Empty(t *testing.T) {
	p := &SSHTunnelProcess{
		Env:  map[string]string{},
		vars: map[string]string{},
	}
	env := p.resolveEnvironment()
	if len(env) != 0 {
		t.Errorf("Expected empty environment, got %v", env)
	}
}

func TestPublicKeyFile(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: false,
		Color: false,
	})

	tests := []struct {
		name        string
		keyPath     string
		expectError bool
	}{
		{
			name:        "valid key file",
			keyPath:     "../../testdata/keys/runp",
			expectError: false,
		},
		{
			name:        "non-existent file",
			keyPath:     "/nonexistent/path/to/key",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authMethod, err := publicKeyFile(tt.keyPath)
			if tt.expectError {
				if err == nil {
					t.Error("publicKeyFile() should return an error for invalid file")
				}
				if authMethod != nil {
					t.Error("publicKeyFile() should return nil auth method on error")
				}
			} else {
				if err != nil {
					t.Errorf("publicKeyFile() unexpected error: %v", err)
				}
				if authMethod == nil {
					t.Error("publicKeyFile() should return non-nil AuthMethod for valid file")
				}
			}
		})
	}
}

func TestSSHTunnelProcess_SetPreconditions(t *testing.T) {
	p := &SSHTunnelProcess{}
	preconditions := Preconditions{
		Os: OsPrecondition{},
	}
	p.SetPreconditions(preconditions)
	// Verifica che le preconditions siano state impostate
	result := p.VerifyPreconditions()
	// Se non ci sono preconditions impostate, dovrebbe restituire Proceed
	if result.Vote != Proceed && result.Vote != Stop {
		t.Errorf("Expected Vote to be Proceed or Stop, got %v", result.Vote)
	}
}

func TestSSHTunnelProcess_executeCmd(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})

	t.Run("error on missing config", func(t *testing.T) {
		p := &SSHTunnelProcess{
			Jump: Endpoint{Host: "invalid-host", Port: 22},
			vars: map[string]string{},
		}
		// This will fail because resolveSSHCommandConfiguration will fail
		// (no auth method configured)
		_, err := p.executeCmd("echo test")
		if err == nil {
			t.Error("Expected error when no auth method is configured")
		}
	})

	t.Run("error on invalid host", func(t *testing.T) {
		p := &SSHTunnelProcess{
			Jump: Endpoint{Host: "invalid-host-that-does-not-exist", Port: 22},
			vars: map[string]string{},
			Auth: Auth{
				Secret: "test-secret",
			},
			User: "testuser",
		}
		// This will fail because SSH connection will fail
		_, err := p.executeCmd("echo test")
		if err == nil {
			t.Error("Expected error when connecting to invalid host")
		}
	})
}

// TestResolveSSHCommandConfiguration_InvalidIdentityFile verifies that
// resolveSSHCommandConfiguration returns an error when the identity file
// contains invalid key data, rather than silently appending a nil
// ssh.AuthMethod alongside other methods. A nil ssh.AuthMethod causes a
// panic when the SSH client iterates auth methods at connection time.
func TestResolveSSHCommandConfiguration_InvalidIdentityFile(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{Debug: false, Color: false})

	f, err := os.CreateTemp("", "invalid_ssh_key_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("this is not a valid PEM-encoded private key")
	f.Close()

	// Use a real known_hosts so we can get past that check and observe
	// that the nil auth method from the invalid identity file is the issue.
	knownHosts, err := os.CreateTemp("", "test_known_hosts_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(knownHosts.Name())
	knownHosts.Close()

	// Provide a valid Secret alongside the broken IdentityFile. Without the
	// fix, publicKeyFile() returns nil which is appended to authMethods, the
	// Secret method keeps len > 0, no error is returned, and the nil element
	// causes a panic at connect time.
	p := &SSHTunnelProcess{
		Auth: Auth{
			IdentityFile: f.Name(),
			Secret:       "some-password",
		},
		KnownHostsFile: knownHosts.Name(),
		vars:           map[string]string{},
	}

	_, err = p.resolveSSHCommandConfiguration()
	if err == nil {
		t.Error("expected error for invalid identity file, got nil — nil ssh.AuthMethod is silently included and will panic at connection time")
	}
}
