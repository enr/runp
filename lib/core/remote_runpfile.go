package core

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	// maxRemoteRunpfileSize caps the size of a downloaded Runpfile.
	maxRemoteRunpfileSize = 1 << 20 // 1 MiB
	remoteFetchTimeout    = 30 * time.Second
)

var checksumRegexp = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// remoteHTTPClient is used to download remote Runpfiles; tests may replace it.
var remoteHTTPClient = &http.Client{Timeout: remoteFetchTimeout}

// FetchRunpfile downloads rawURL to a temporary file, verifies the SHA-256
// checksum when checksum is non-empty (format "sha256:<hex>"), and returns the
// path of the temp file. The caller must remove it with os.Remove.
//
// A Runpfile is executable configuration, so a plain http:// URL is accepted
// only together with a checksum: without TLS nothing else protects it from
// being tampered with in transit.
func FetchRunpfile(rawURL, checksum string) (string, error) {
	want := strings.ToLower(strings.TrimSpace(checksum))
	if want != "" && !checksumRegexp.MatchString(want) {
		return "", fmt.Errorf("invalid checksum %q: expected format sha256:<64 hex characters>", checksum)
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid Runpfile URL %s: %w", rawURL, err)
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
	case "http":
		if want == "" {
			return "", fmt.Errorf("refusing to fetch Runpfile over plain HTTP without --checksum: %s (use https:// or pass --checksum sha256:<hex>)", rawURL)
		}
	default:
		return "", fmt.Errorf("unsupported URL scheme %q for Runpfile: %s", u.Scheme, rawURL)
	}

	resp, err := remoteHTTPClient.Get(rawURL) //nolint:gosec // URL is user-supplied via --file
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
	// Read one byte past the limit to detect oversized files.
	n, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(resp.Body, maxRemoteRunpfileSize+1))
	f.Close()
	if err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("writing downloaded Runpfile: %w", err)
	}
	if n > maxRemoteRunpfileSize {
		os.Remove(tmpPath)
		return "", fmt.Errorf("remote Runpfile %s exceeds the %d bytes limit", rawURL, maxRemoteRunpfileSize)
	}

	if want != "" {
		got := fmt.Sprintf("sha256:%x", h.Sum(nil))
		if got != want {
			os.Remove(tmpPath)
			return "", fmt.Errorf("checksum mismatch for %s:\n  want %s\n  got  %s", rawURL, want, got)
		}
	}
	return tmpPath, nil
}
