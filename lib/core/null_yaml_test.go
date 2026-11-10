package core

import (
	"os"
	"path/filepath"
	"testing"
)

// A YAML document that is null ("null", "~") or empty must not panic.
func TestLoadRunpfileNullDocument(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{})
	for _, content := range []string{"null\n", "~\n", ""} {
		p := filepath.Join(t.TempDir(), "Runpfile")
		if err := os.WriteFile(p, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		rf, err := LoadRunpfileFromPath(p)
		if err != nil {
			continue
		}
		if rf == nil {
			t.Fatalf("content %q: nil Runpfile without error", content)
		}
		if valid, _ := IsRunpfileValid(rf); valid {
			t.Errorf("content %q: Runpfile without units reported valid", content)
		}
	}
}

func TestSettingsNullDocument(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{})
	for _, content := range []string{"null\n", "~\n", ""} {
		es := settingsFromBytes([]byte(content), &EnvironmentSettings{ContainerRunnerExe: "docker"})
		if es == nil || es.ContainerRunnerExe != "docker" {
			t.Errorf("content %q: got %+v, want the defaults", content, es)
		}
	}
}
