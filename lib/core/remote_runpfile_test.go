package core

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

const testRunpfileYAML = `units:
  web:
    host:
      cmd: echo hello
`

// useTLSServer serves handler over HTTPS and makes FetchRunpfile trust it.
func useTLSServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewTLSServer(handler)
	orig := remoteHTTPClient
	remoteHTTPClient = srv.Client()
	t.Cleanup(func() {
		remoteHTTPClient = orig
		srv.Close()
	})
	return srv
}

func TestFetchRunpfile_Success(t *testing.T) {
	srv := useTLSServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testRunpfileYAML))
	})

	path, err := FetchRunpfile(srv.URL, "")
	if err != nil {
		t.Fatalf("FetchRunpfile: unexpected error: %v", err)
	}
	defer os.Remove(path)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading temp file: %v", err)
	}
	if string(data) != testRunpfileYAML {
		t.Errorf("content mismatch:\n  want %q\n  got  %q", testRunpfileYAML, string(data))
	}
}

func TestFetchRunpfile_ChecksumOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testRunpfileYAML))
	}))
	defer srv.Close()

	h := sha256.Sum256([]byte(testRunpfileYAML))
	checksum := fmt.Sprintf("sha256:%x", h)

	path, err := FetchRunpfile(srv.URL, checksum)
	if err != nil {
		t.Fatalf("FetchRunpfile with correct checksum: unexpected error: %v", err)
	}
	defer os.Remove(path)
}

func TestFetchRunpfile_ChecksumMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testRunpfileYAML))
	}))
	defer srv.Close()

	_, err := FetchRunpfile(srv.URL, "sha256:"+strings.Repeat("0", 64))
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("FetchRunpfile: expected checksum mismatch error, got %v", err)
	}
}

func TestFetchRunpfile_HTTP404(t *testing.T) {
	srv := useTLSServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	_, err := FetchRunpfile(srv.URL, "")
	if err == nil {
		t.Fatal("FetchRunpfile: expected error for HTTP 404, got nil")
	}
}

func TestFetchRunpfile_PlainHTTPRequiresChecksum(t *testing.T) {
	requested := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = true
		w.Write([]byte(testRunpfileYAML))
	}))
	defer srv.Close()

	_, err := FetchRunpfile(srv.URL, "")
	if err == nil || !strings.Contains(err.Error(), "plain HTTP") {
		t.Fatalf("expected plain HTTP refusal, got %v", err)
	}
	if requested {
		t.Error("the URL must not be fetched when it is refused")
	}
}

func TestFetchRunpfile_InvalidChecksumFormat(t *testing.T) {
	_, err := FetchRunpfile("https://127.0.0.1:1/Runpfile", "sha256:deadbeef")
	if err == nil || !strings.Contains(err.Error(), "invalid checksum") {
		t.Fatalf("expected invalid checksum error, got %v", err)
	}
}

func TestFetchRunpfile_UnsupportedScheme(t *testing.T) {
	_, err := FetchRunpfile("file:///etc/passwd", "")
	if err == nil || !strings.Contains(err.Error(), "unsupported URL scheme") {
		t.Fatalf("expected unsupported scheme error, got %v", err)
	}
}

func TestFetchRunpfile_TooLarge(t *testing.T) {
	srv := useTLSServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Repeat("#", maxRemoteRunpfileSize+1)))
	})
	_, err := FetchRunpfile(srv.URL, "")
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected size limit error, got %v", err)
	}
}
