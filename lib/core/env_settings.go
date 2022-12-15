package core

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/enr/go-files/files"
	"github.com/mitchellh/go-homedir"
	yaml "gopkg.in/yaml.v3"
)

// EnvironmentSettings represents settings for the current box.
type EnvironmentSettings struct {
	ContainerRunnerExe string `yaml:"container_runner"`
}

func loadEnvironmentSettings() *EnvironmentSettings {
	var es = &EnvironmentSettings{
		ContainerRunnerExe: "docker",
	}
	esPath, err := environmentSettingsPath()
	if err != nil {
		ui.WriteLinef(`Error resolving settings path: %v`, err)
		return es
	}
	if !files.Exists(esPath) {
		ui.WriteLinef(`Env settings file not found: %s`, esPath)
		ui.Debugf(`Using default env settings`)
		return es
	}
	if !files.IsRegular(esPath) {
		ui.WriteLinef(`Invalid env settings file: %s`, esPath)
		return es
	}
	ui.Debugf(`Loading env settings from file %s`, esPath)
	f, err := os.ReadFile(esPath)
	if err != nil {
		ui.WriteLinef(`Error loading env settings %s: %v`, esPath, err)
		return es
	}
	return settingsFromBytes(f, es)
}

func settingsFromBytes(f []byte, es *EnvironmentSettings) *EnvironmentSettings {
	if err := yaml.Unmarshal(f, &es); err != nil {
		ui.WriteLinef(`Error parsing env settings file: %v`, err)
		return es
	}
	fmt.Printf("%+v\n", es)
	return es
}

func environmentSettingsPath() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	return filepath.FromSlash(path.Join(home, `.runp`, `settings.yaml`)), nil
}
