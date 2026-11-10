package core

import (
	"os"
	"path/filepath"
	"testing"
)

func writeRemoteCopy(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "runpfile-123.yml")
	if err := os.WriteFile(p, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}

// A remote Runpfile is downloaded to a temporary file: its root is the
// current directory, not the temporary directory, and its identity (PID
// files, container label) comes from its URL.
func TestLoadRemoteRunpfileIdentity(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{})
	spec := "units:\n  a:\n    host:\n      command: \"true\"\n      workdir: sub\n"
	wd, _ := os.Getwd()

	one, err := LoadRemoteRunpfile(writeRemoteCopy(t, spec), "https://example.com/one.yml")
	if err != nil {
		t.Fatal(err)
	}
	two, err := LoadRemoteRunpfile(writeRemoteCopy(t, spec), "https://example.com/two.yml")
	if err != nil {
		t.Fatal(err)
	}
	again, err := LoadRemoteRunpfile(writeRemoteCopy(t, spec), "https://example.com/one.yml")
	if err != nil {
		t.Fatal(err)
	}
	if one.Root != wd {
		t.Errorf("root = %s, want the current directory %s", one.Root, wd)
	}
	if got := one.Units["a"].Host.WorkingDir; got != filepath.Join(wd, "sub") {
		t.Errorf("workdir = %s, want it relative to the current directory", got)
	}
	if one.ProjectID() == two.ProjectID() {
		t.Error("two remote Runpfiles share the same project ID")
	}
	if one.ProjectID() != again.ProjectID() {
		t.Error("the same URL must keep the same project ID across runs")
	}
	if one.ProjectID() == ProjectID(wd) {
		t.Error("a remote Runpfile shares the project ID of a local Runpfile in the current directory")
	}
}

func TestLoadRemoteRunpfileRejectsInclude(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{})
	spec := "include: [other.yml]\nunits:\n  a:\n    host:\n      command: \"true\"\n"
	if _, err := LoadRemoteRunpfile(writeRemoteCopy(t, spec), "https://example.com/x.yml"); err == nil {
		t.Error("expected an error for include in a remote Runpfile")
	}
}
