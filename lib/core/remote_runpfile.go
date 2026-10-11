package core

import (
	"crypto/sha256"
	"errors"
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
	want, err := checkRemoteSource(rawURL, checksum)
	if err != nil {
		return "", err
	}

	client := *remoteHTTPClient
	if want == "" {
		// Without a checksum only TLS protects the download: never follow a
		// redirect to plain HTTP.
		client.CheckRedirect = httpsOnlyRedirect
	}
	resp, err := client.Get(rawURL) //nolint:gosec // URL is user-supplied via --file
	if err != nil {
		return "", fmt.Errorf("fetching Runpfile from %s: %w", rawURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetching Runpfile from %s: HTTP %d", rawURL, resp.StatusCode)
	}

	tmpPath, got, err := downloadToTemp(rawURL, resp.Body)
	if err != nil {
		return "", err
	}
	if want != "" && got != want {
		os.Remove(tmpPath)
		return "", fmt.Errorf("checksum mismatch for %s:\n  want %s\n  got  %s", rawURL, want, got)
	}
	return tmpPath, nil
}

// checkRemoteSource validates the checksum format and the URL scheme, and
// returns the normalized checksum.
func checkRemoteSource(rawURL, checksum string) (string, error) {
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
	return want, nil
}

func httpsOnlyRedirect(req *http.Request, via []*http.Request) error {
	if !strings.EqualFold(req.URL.Scheme, "https") {
		return fmt.Errorf("refusing redirect to %s without --checksum: only https:// is allowed", req.URL.Redacted())
	}
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}
	return nil
}

// downloadToTemp copies body to a temporary file, enforcing the size limit,
// and returns its path and its "sha256:<hex>" checksum.
func downloadToTemp(rawURL string, body io.Reader) (string, string, error) {
	f, err := os.CreateTemp("", "runpfile-*.yml")
	if err != nil {
		return "", "", fmt.Errorf("creating temp file for remote Runpfile: %w", err)
	}
	tmpPath := f.Name()

	h := sha256.New()
	// Read one byte past the limit to detect oversized files.
	n, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(body, maxRemoteRunpfileSize+1))
	f.Close()
	if err != nil {
		os.Remove(tmpPath)
		return "", "", fmt.Errorf("writing downloaded Runpfile: %w", err)
	}
	if n > maxRemoteRunpfileSize {
		os.Remove(tmpPath)
		return "", "", fmt.Errorf("remote Runpfile %s exceeds the %d bytes limit", rawURL, maxRemoteRunpfileSize)
	}
	return tmpPath, fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}
