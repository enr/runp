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
// The file also records the identity (start time) of the process, so that a
// stale file whose PID has been reused by another process can be detected.
func WritePIDFile(pidDir, unitName string, pid int) error {
	if err := os.MkdirAll(pidDir, 0755); err != nil {
		return err
	}
	content := strconv.Itoa(pid) + "\n"
	if identity := processIdentity(pid); identity != "" {
		content += identity + "\n"
	}
	return os.WriteFile(pidFilePath(pidDir, unitName), []byte(content), 0644)
}

// ReadPIDFile reads the PID stored in <pidDir>/<unitName>.pid.
func ReadPIDFile(pidDir, unitName string) (int, error) {
	pid, _, err := readPIDRecord(pidDir, unitName)
	return pid, err
}

// readPIDRecord returns the PID and the process identity stored in the PID
// file; identity is empty for files written without one.
func readPIDRecord(pidDir, unitName string) (int, string, error) {
	data, err := os.ReadFile(pidFilePath(pidDir, unitName))
	if err != nil {
		return 0, "", err
	}
	lines := strings.SplitN(strings.TrimSpace(string(data)), "\n", 2)
	pid, err := strconv.Atoi(strings.TrimSpace(lines[0]))
	if err != nil {
		return 0, "", err
	}
	identity := ""
	if len(lines) > 1 {
		identity = strings.TrimSpace(lines[1])
	}
	return pid, identity, nil
}

// pidFileState describes the process referenced by a PID file.
type pidFileState int

const (
	pidFileRunning pidFileState = iota
	// pidFileExited means the process is no longer running.
	pidFileExited
	// pidFileReused means the PID now belongs to a different process.
	pidFileReused
)

// probePIDFile reads the unit's PID file and reports whether the process it
// references is still the one runp started.
func probePIDFile(pidDir, unitName string) (int, pidFileState, error) {
	pid, identity, err := readPIDRecord(pidDir, unitName)
	if err != nil {
		return 0, pidFileExited, err
	}
	if !isProcessAlive(pid) {
		return pid, pidFileExited, nil
	}
	if identity != "" {
		if current := processIdentity(pid); current != "" && current != identity {
			return pid, pidFileReused, nil
		}
	}
	return pid, pidFileRunning, nil
}

// RemovePIDFile deletes the PID file for the given unit, ignoring errors.
func RemovePIDFile(pidDir, unitName string) {
	os.Remove(pidFilePath(pidDir, unitName))
}

// removePIDFileIfOwned deletes the PID file only if it still refers to pid.
// A unit restarted by "runp reload" writes a new PID file: the exiting
// instance must not delete it.
func removePIDFileIfOwned(pidDir, unitName string, pid int) {
	if current, err := ReadPIDFile(pidDir, unitName); err == nil && current == pid {
		RemovePIDFile(pidDir, unitName)
	}
}

func pidFilePath(pidDir, unitName string) string {
	return filepath.Join(pidDir, unitName+".pid")
}
