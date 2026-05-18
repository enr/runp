package core

import (
	"strings"
	"testing"
)

func TestListEntriesSorted(t *testing.T) {
	ui = CreateMainLogger(" ", 6, "TEST", false, false)

	rf := &Runpfile{
		Units: map[string]*RunpUnit{
			"zoo":   {Name: "zoo", Host: &HostProcess{}},
			"alpha": {Name: "alpha", Container: &ContainerProcess{Image: "nginx"}},
			"beta": {
				Name:        "beta",
				Description: "a tunnel",
				SSHTunnel: &SSHTunnelProcess{
					Local:  Endpoint{Host: "localhost", Port: 8080},
					Jump:   Endpoint{Host: "jump.example.com", Port: 22},
					Target: Endpoint{Host: "db.internal", Port: 5432},
				},
			},
		},
	}

	entries := ListEntries(rf)

	if len(entries) != 3 {
		t.Fatalf("Expected 3 entries, got %d", len(entries))
	}
	names := []string{entries[0].Name, entries[1].Name, entries[2].Name}
	expected := []string{"alpha", "beta", "zoo"}
	for i, n := range names {
		if n != expected[i] {
			t.Errorf("entries[%d].Name = %q, want %q", i, n, expected[i])
		}
	}

	if entries[1].Description != "a tunnel" {
		t.Errorf("Expected description 'a tunnel', got %q", entries[1].Description)
	}
	if entries[1].Kind != "ssh_tunnel" {
		t.Errorf("Expected kind 'ssh_tunnel', got %q", entries[1].Kind)
	}
}

func TestPreconditionSummaryNone(t *testing.T) {
	s := preconditionSummary(Preconditions{})
	if s != "-" {
		t.Errorf("Expected '-' for empty preconditions, got %q", s)
	}
}

func TestPreconditionSummaryOs(t *testing.T) {
	p := Preconditions{
		Os: OsPrecondition{Inclusions: []string{"linux", "darwin"}},
	}
	s := preconditionSummary(p)
	if !strings.HasPrefix(s, "os:") {
		t.Errorf("Expected summary to start with 'os:', got %q", s)
	}
	if !strings.Contains(s, "linux") || !strings.Contains(s, "darwin") {
		t.Errorf("Expected summary to mention 'linux' and 'darwin', got %q", s)
	}
}

func TestPreconditionSummaryEnvVars(t *testing.T) {
	p := Preconditions{
		EnvVars: EnvVarsPrecondition{
			EnvVars: []EnvVarCheck{
				{Name: "DATABASE_URL", Condition: EnvVarConditionIsSet},
				{Name: "SECRET_KEY", Condition: EnvVarConditionIsSet},
			},
		},
	}
	s := preconditionSummary(p)
	if !strings.Contains(s, "env:") {
		t.Errorf("Expected summary to contain 'env:', got %q", s)
	}
	if !strings.Contains(s, "DATABASE_URL") {
		t.Errorf("Expected summary to mention DATABASE_URL, got %q", s)
	}
}

func TestPreconditionSummaryRunpVersion(t *testing.T) {
	p := Preconditions{
		Runp: RunpVersionPrecondition{
			Operator: GreaterThanOrEqual,
			Version:  "0.10.0",
		},
	}
	s := preconditionSummary(p)
	if !strings.Contains(s, "runp:>=0.10.0") {
		t.Errorf("Expected 'runp:>=0.10.0', got %q", s)
	}
}
