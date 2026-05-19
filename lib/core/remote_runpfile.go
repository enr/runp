package core

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// FetchRunpfile downloads rawURL to a temporary file, verifies the SHA-256
// checksum when checksum is non-empty (format "sha256:<lowercase-hex>"), and
// returns the path of the temp file. The caller must remove it with os.Remove.
func FetchRunpfile(rawURL, checksum string) (string, error) {
	resp, err := http.Get(rawURL) //nolint:gosec // URL is user-supplied via --file
	if err != nil {
		return "", fmt.Errorf("fetching Runpfile from %s: %w", rawURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetching Runpfile from %s: HTTP %d", rawURL, resp.StatusCode)
	}

	f, err := os.CreateTemp("", "runpfile-*.yml")
	if err != nil {
		return "", fmt.Errorf("creating temp file for remote Runpfile: %w", err)
	}
	tmpPath := f.Name()

	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(f, h), resp.Body); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("writing downloaded Runpfile: %w", err)
	}
	f.Close()

	if checksum != "" {
		got := fmt.Sprintf("sha256:%x", h.Sum(nil))
		want := strings.ToLower(checksum)
		if got != want {
			os.Remove(tmpPath)
			return "", fmt.Errorf("checksum mismatch for %s:\n  want %s\n  got  %s", rawURL, want, got)
		}
	}
	return tmpPath, nil
}
