package core

import (
	"errors"
	"testing"
)

func TestParseStopTimeout(t *testing.T) {
	tests := []struct {
		input    string
		wantSecs float64
	}{
		{"10s", 10},
		{"1m", 60},
		{"500ms", 0.5},
		{"", 5},
		{"invalid", 5},
	}
	for _, tc := range tests {
		got := parseStopTimeout(tc.input)
		if got.Seconds() != tc.wantSecs {
			t.Errorf("parseStopTimeout(%q) = %v, want %v", tc.input, got.Seconds(), tc.wantSecs)
		}
	}
}

func TestStopUnitSSHTunnelReturnsNotSupported(t *testing.T) {
	ui = CreateMainLogger(" ", 6, "TEST", false, false)

	unit := &RunpUnit{
		Name:      "my-tunnel",
		SSHTunnel: &SSHTunnelProcess{},
	}
	err := StopUnit(unit, "", nil)
	if err == nil {
		t.Fatal("Expected error for SSH tunnel unit, got nil")
	}
	if !errors.Is(err, ErrReloadNotSupported) {
		t.Errorf("Expected ErrReloadNotSupported, got: %v", err)
	}
}

func TestStopUnitNoPIDFileReturnsError(t *testing.T) {
	ui = CreateMainLogger(" ", 6, "TEST", false, false)

	dir := t.TempDir()
	unit := &RunpUnit{
		Name: "my-host",
		Host: &HostProcess{CommandLine: "echo hi"},
	}
	err := StopUnit(unit, dir, nil)
	if err == nil {
		t.Fatal("Expected error when no PID file exists, got nil")
	}
	if errors.Is(err, ErrReloadNotSupported) {
		t.Errorf("Expected a non-running error, got ErrReloadNotSupported")
	}
}

func TestStopUnitUnknownTypeReturnsError(t *testing.T) {
	ui = CreateMainLogger(" ", 6, "TEST", false, false)

	unit := &RunpUnit{Name: "ghost"}
	err := StopUnit(unit, "", nil)
	if err == nil {
		t.Fatal("Expected error for unit with no process type, got nil")
	}
}
