package core

import (
	"fmt"
	"strings"
	"testing"
)

func TestDefaults(t *testing.T) {
	expectedContainerRunner := `docker`
	contents := []byte(``)
	defaultSettings := &EnvironmentSettings{
		ContainerRunnerExe: expectedContainerRunner,
	}
	es := settingsFromBytes(contents, defaultSettings)
	if es.ContainerRunnerExe != expectedContainerRunner {
		t.Errorf("Expected ContainerRunnerExe '%s', got '%s' %+v", expectedContainerRunner, es.ContainerRunnerExe, es)
	}
}

func TestCustomSettings(t *testing.T) {
	expectedContainerRunner := `/path/to/podman`
	contents := []byte(fmt.Sprintf(`container_runner: %s`, expectedContainerRunner))
	defaultSettings := &EnvironmentSettings{
		ContainerRunnerExe: `docker`,
	}
	es := settingsFromBytes(contents, defaultSettings)
	if es.ContainerRunnerExe != expectedContainerRunner {
		t.Errorf("Expected ContainerRunnerExe '%s', got '%s' %+v", expectedContainerRunner, es.ContainerRunnerExe, es)
	}
}

func TestEnvironmentSettingsPath(t *testing.T) {
	// Test environmentSettingsPath
	path, err := environmentSettingsPath()
	if err != nil {
		t.Errorf("environmentSettingsPath should not return error, got %v", err)
	}
	if path == "" {
		t.Error("environmentSettingsPath should return a non-empty path")
	}

	// Verify that the path contains .runp/settings.yaml
	if !strings.Contains(path, ".runp") {
		t.Errorf("Expected path to contain '.runp', got '%s'", path)
	}
	if !strings.Contains(path, "settings.yaml") {
		t.Errorf("Expected path to contain 'settings.yaml', got '%s'", path)
	}
}
