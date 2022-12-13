package core

import (
	"path"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
)

// EnvironmentSettings represents settings for the current box.
type EnvironmentSettings struct {
	ContainerRunnerExe string `yaml:"container_runner"`
}

func defaultSettings() *EnvironmentSettings {
	return &EnvironmentSettings{
		ContainerRunnerExe: `docker`,
	}
}

func environmentSettingsPath() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	return filepath.FromSlash(path.Join(home, `.runp`, `settings.yaml`)), nil
}
