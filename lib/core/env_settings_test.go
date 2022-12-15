package core

import (
	"fmt"
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
