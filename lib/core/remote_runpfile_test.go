package core

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

const testRunpfileYAML = `units:
  web:
    host:
      cmd: echo hello
`

func TestFetchRunpfile_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testRunpfileYAML))
	}))
	defer srv.Close()

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

	_, err := FetchRunpfile(srv.URL, "sha256:deadbeef")
	if err == nil {
		t.Fatal("FetchRunpfile: expected checksum mismatch error, got nil")
	}
}

func TestFetchRunpfile_HTTP404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	_, err := FetchRunpfile(srv.URL, "")
	if err == nil {
		t.Fatal("FetchRunpfile: expected error for HTTP 404, got nil")
	}
}
