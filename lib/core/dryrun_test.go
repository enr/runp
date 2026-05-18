package core

import (
	"testing"
)

func TestDryRunPreviews(t *testing.T) {
	ui = CreateMainLogger(" ", 6, "TEST", false, false)

	rf := &Runpfile{
		Vars: map[string]string{"greeting": "hello"},
		Units: map[string]*RunpUnit{
			"web": {
				Name: "web",
				Host: &HostProcess{CommandLine: "echo {{vars greeting}}"},
			},
			"db": {
				Name:      "db",
				Container: &ContainerProcess{Image: "postgres:15"},
			},
		},
	}
	executor := NewExecutor(rf)
	previews := executor.DryRunPreviews()

	if len(previews) != 2 {
		t.Fatalf("Expected 2 previews, got %d", len(previews))
	}

	// Previews are sorted by name: "db" < "web"
	db := previews[0]
	if db.Name != "db" {
		t.Errorf("Expected first preview to be 'db', got %q", db.Name)
	}
	if db.Kind != "container" {
		t.Errorf("Expected kind 'container', got %q", db.Kind)
	}
	if db.Command != "postgres:15" {
		t.Errorf("Expected command 'postgres:15', got %q", db.Command)
	}

	web := previews[1]
	if web.Name != "web" {
		t.Errorf("Expected second preview to be 'web', got %q", web.Name)
	}
	if web.Kind != "host" {
		t.Errorf("Expected kind 'host', got %q", web.Kind)
	}
	// Variable substitution should have resolved {{vars greeting}} -> "hello"
	if web.Command != "echo hello" {
		t.Errorf("Expected resolved command 'echo hello', got %q", web.Command)
	}
	if web.Skipped {
		t.Error("Expected web unit not to be skipped")
	}
}

func TestDryRunPreviewsSkippedUnit(t *testing.T) {
	ui = CreateMainLogger(" ", 6, "TEST", false, false)

	rf := &Runpfile{
		Units: map[string]*RunpUnit{
			"unix-only": {
				Name: "unix-only",
				Host: &HostProcess{CommandLine: "echo hi"},
				Preconditions: Preconditions{
					Os: OsPrecondition{
						Inclusions: []string{"definitely-not-this-os"},
					},
				},
			},
		},
	}
	executor := NewExecutor(rf)
	previews := executor.DryRunPreviews()

	if len(previews) != 1 {
		t.Fatalf("Expected 1 preview, got %d", len(previews))
	}
	if !previews[0].Skipped {
		t.Error("Expected unit to be marked as skipped")
	}
	if previews[0].SkipReason == "" {
		t.Error("Expected a skip reason")
	}
}

func TestDryRunPreviewsSSHTunnel(t *testing.T) {
	ui = CreateMainLogger(" ", 6, "TEST", false, false)

	rf := &Runpfile{
		Units: map[string]*RunpUnit{
			"tunnel": {
				Name: "tunnel",
				SSHTunnel: &SSHTunnelProcess{
					Local:  Endpoint{Host: "localhost", Port: 8080},
					Jump:   Endpoint{Host: "jump.example.com", Port: 22},
					Target: Endpoint{Host: "db.internal", Port: 5432},
				},
			},
		},
	}
	executor := NewExecutor(rf)
	previews := executor.DryRunPreviews()

	if len(previews) != 1 {
		t.Fatalf("Expected 1 preview, got %d", len(previews))
	}
	p := previews[0]
	if p.Kind != "ssh_tunnel" {
		t.Errorf("Expected kind 'ssh_tunnel', got %q", p.Kind)
	}
	if p.Command == "" {
		t.Error("Expected non-empty command summary for SSH tunnel")
	}
}
