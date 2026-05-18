package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mitchellh/go-homedir"
)

func TestGetSettingValueUnknownKey(t *testing.T) {
	ui = CreateMainLogger(" ", 6, "TEST", false, false)
	_, err := GetSettingValue("nonexistent_key")
	if err == nil {
		t.Fatal("Expected error for unknown key, got nil")
	}
}

func TestGetSettingValueDefault(t *testing.T) {
	ui = CreateMainLogger(" ", 6, "TEST", false, false)
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	homedir.Reset()
	t.Cleanup(homedir.Reset)

	val, err := GetSettingValue("container_runner")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if val != "docker" {
		t.Errorf("Expected default %q, got %q", "docker", val)
	}
}

func TestSetAndGetSettingValue(t *testing.T) {
	ui = CreateMainLogger(" ", 6, "TEST", false, false)
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	homedir.Reset()
	t.Cleanup(homedir.Reset)

	if err := SetSettingValue("container_runner", "podman"); err != nil {
		t.Fatalf("SetSettingValue failed: %v", err)
	}

	val, err := GetSettingValue("container_runner")
	if err != nil {
		t.Fatalf("GetSettingValue failed: %v", err)
	}
	if val != "podman" {
		t.Errorf("Expected %q, got %q", "podman", val)
	}

	// Confirm the file was created at the expected path.
	settingsPath := filepath.Join(dir, ".runp", "settings.yaml")
	if _, err := os.Stat(settingsPath); err != nil {
		t.Errorf("Expected settings file at %s, stat failed: %v", settingsPath, err)
	}
}

func TestSetSettingValueUnknownKey(t *testing.T) {
	ui = CreateMainLogger(" ", 6, "TEST", false, false)
	err := SetSettingValue("unknown_key", "value")
	if err == nil {
		t.Fatal("Expected error for unknown key, got nil")
	}
}

func TestSupportedSettingKeysHelp(t *testing.T) {
	help := SupportedSettingKeysHelp()
	if len(help) == 0 {
		t.Error("Expected non-empty help string")
	}
	if help[:len("  container_runner")] != "  container_runner" {
		t.Errorf("Expected help to start with container_runner key, got: %s", help[:40])
	}
}
