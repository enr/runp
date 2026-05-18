package core

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mitchellh/go-homedir"
)

// PIDDirForRoot returns the directory used to store PID files for a given Runpfile root.
// The directory is scoped by a hash of the root path to avoid collisions across projects.
func PIDDirForRoot(root string) (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	h := sha256.Sum256([]byte(root))
	hash := fmt.Sprintf("%x", h)[:8]
	return filepath.Join(home, ".runp", "pids", hash), nil
}

// WritePIDFile writes pid to <pidDir>/<unitName>.pid, creating the directory if needed.
func WritePIDFile(pidDir, unitName string, pid int) error {
	if err := os.MkdirAll(pidDir, 0755); err != nil {
		return err
	}
	p := filepath.Join(pidDir, unitName+".pid")
	return os.WriteFile(p, []byte(strconv.Itoa(pid)), 0644)
}

// ReadPIDFile reads the PID stored in <pidDir>/<unitName>.pid.
func ReadPIDFile(pidDir, unitName string) (int, error) {
	p := filepath.Join(pidDir, unitName+".pid")
	data, err := os.ReadFile(p)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

// RemovePIDFile deletes the PID file for the given unit, ignoring errors.
func RemovePIDFile(pidDir, unitName string) {
	os.Remove(filepath.Join(pidDir, unitName+".pid"))
}
