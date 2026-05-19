package e2e

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestURLFile(t *testing.T) {
	content, err := os.ReadFile("../testdata/runpfiles/env.yml")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))
	defer srv.Close()

	hash := fmt.Sprintf("%x", sha256.Sum256(content))

	t.Run("url_no_checksum", func(t *testing.T) {
		out, code := runp(t, "list", "--file", srv.URL)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d\noutput: %s", code, out)
		}
		if !strings.Contains(out, "env-test-unit") {
			t.Errorf("expected unit name in output, got:\n%s", out)
		}
	})

	t.Run("url_checksum_ok", func(t *testing.T) {
		out, code := runp(t, "list", "--file", srv.URL, "--checksum", "sha256:"+hash)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d\noutput: %s", code, out)
		}
		if !strings.Contains(out, "env-test-unit") {
			t.Errorf("expected unit name in output, got:\n%s", out)
		}
	})

	t.Run("url_checksum_mismatch", func(t *testing.T) {
		out, code := runp(t, "list", "--file", srv.URL, "--checksum", "sha256:deadbeef")
		if code != 2 {
			t.Fatalf("expected exit 2, got %d\noutput: %s", code, out)
		}
		if !strings.Contains(strings.ToLower(out), "checksum mismatch") {
			t.Errorf("expected 'checksum mismatch' in output, got:\n%s", out)
		}
	})

	t.Run("url_unreachable", func(t *testing.T) {
		out, code := runp(t, "list", "--file", "http://127.0.0.1:1")
		if code != 2 {
			t.Fatalf("expected exit 2, got %d\noutput: %s", code, out)
		}
		lower := strings.ToLower(out)
		if !strings.Contains(lower, "fetching") && !strings.Contains(lower, "connection refused") {
			t.Errorf("expected fetch error in output, got:\n%s", out)
		}
	})
}

func runp(t *testing.T, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command("../bin/runp", args...)
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		} else {
			t.Fatalf("exec error: %v", err)
		}
	}
	return string(out), code
}
