package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// SettingMeta describes a settings key: its human-readable description and default value.
type SettingMeta struct {
	Description string
	Default     string
}

// settingsKeyMeta is the authoritative list of supported settings keys.
var settingsKeyMeta = map[string]SettingMeta{
	"container_runner": {
		Description: "Container runtime executable used for container units (docker or podman)",
		Default:     "docker",
	},
}

// EnvironmentSettingsPath returns the absolute path to ~/.runp/settings.yaml.
func EnvironmentSettingsPath() (string, error) {
	return environmentSettingsPath()
}

// SupportedSettingKeys returns the map of supported settings keys and their metadata.
func SupportedSettingKeys() map[string]SettingMeta {
	return settingsKeyMeta
}

// SupportedSettingKeysHelp returns a formatted string listing all supported keys.
func SupportedSettingKeysHelp() string {
	keys := make([]string, 0, len(settingsKeyMeta))
	for k := range settingsKeyMeta {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		m := settingsKeyMeta[k]
		fmt.Fprintf(&b, "  %-20s  %s (default: %q)\n", k, m.Description, m.Default)
	}
	return b.String()
}

// GetSettingValue returns the current value of a settings key.
// Returns an error for unknown keys.
func GetSettingValue(key string) (string, error) {
	if _, ok := settingsKeyMeta[key]; !ok {
		return "", fmt.Errorf("unknown setting key %q\n\nSupported keys:\n%s", key, SupportedSettingKeysHelp())
	}
	es := loadEnvironmentSettings()
	return settingFieldByKey(es, key), nil
}

// SetSettingValue writes a key=value pair to ~/.runp/settings.yaml.
// The directory is created if it does not exist.
func SetSettingValue(key, value string) error {
	if _, ok := settingsKeyMeta[key]; !ok {
		return fmt.Errorf("unknown setting key %q\n\nSupported keys:\n%s", key, SupportedSettingKeysHelp())
	}
	path, err := environmentSettingsPath()
	if err != nil {
		return fmt.Errorf("resolving settings path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating settings directory: %w", err)
	}
	es := loadEnvironmentSettings()
	applySettingField(es, key, value)
	data, err := yaml.Marshal(es)
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

func settingFieldByKey(es *EnvironmentSettings, key string) string {
	switch key {
	case "container_runner":
		return es.ContainerRunnerExe
	}
	return ""
}

func applySettingField(es *EnvironmentSettings, key, value string) {
	switch key {
	case "container_runner":
		es.ContainerRunnerExe = value
	}
}
