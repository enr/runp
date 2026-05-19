package core

import (
	"testing"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input   string
		want    LogLevel
		wantErr bool
	}{
		{"error", LogLevelError, false},
		{"warn", LogLevelWarn, false},
		{"warning", LogLevelWarn, false},
		{"info", LogLevelInfo, false},
		{"debug", LogLevelDebug, false},
		{"trace", LogLevelTrace, false},
		{"INFO", LogLevelInfo, false}, // case-insensitive
		{"DEBUG", LogLevelDebug, false},
		{"", LogLevelUnset, true},
		{"verbose", LogLevelUnset, true},
		{"unknown", LogLevelUnset, true},
	}
	for _, tc := range tests {
		got, err := ParseLogLevel(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseLogLevel(%q): expected error, got nil", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseLogLevel(%q): unexpected error: %v", tc.input, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseLogLevel(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestLogLevelString(t *testing.T) {
	tests := []struct {
		level LogLevel
		want  string
	}{
		{LogLevelError, "error"},
		{LogLevelWarn, "warn"},
		{LogLevelInfo, "info"},
		{LogLevelDebug, "debug"},
		{LogLevelTrace, "trace"},
		{LogLevelUnset, "info"}, // unset renders as default
	}
	for _, tc := range tests {
		if got := tc.level.String(); got != tc.want {
			t.Errorf("LogLevel(%d).String() = %q, want %q", tc.level, got, tc.want)
		}
	}
}

func TestLoggerLevelFiltering(t *testing.T) {
	// Use a stub logger to verify level-based suppression.
	t.Run("info level suppresses Debugf", func(t *testing.T) {
		l := CreateMainLoggerWithLevel("T", 1, "%s", LogLevelInfo, false)
		// Debugf should be a no-op at info level — just verify no panic.
		n, _ := l.Debugf("should be suppressed")
		if n != 0 {
			t.Errorf("Expected Debugf to return 0 bytes at info level, got %d", n)
		}
	})

	t.Run("debug level allows Debugf", func(t *testing.T) {
		l := CreateMainLoggerWithLevel("T", 1, "%s", LogLevelDebug, false)
		n, _ := l.Debugf("should be written")
		if n == 0 {
			t.Errorf("Expected Debugf to write bytes at debug level, got 0")
		}
	})

	t.Run("warn level suppresses WriteLinef", func(t *testing.T) {
		l := CreateMainLoggerWithLevel("T", 1, "%s", LogLevelWarn, false)
		n, _ := l.WriteLinef("info message")
		if n != 0 {
			t.Errorf("Expected WriteLinef to return 0 bytes at warn level, got %d", n)
		}
	})

	t.Run("info level allows WriteLinef", func(t *testing.T) {
		l := CreateMainLoggerWithLevel("T", 1, "%s", LogLevelInfo, false)
		n, _ := l.WriteLinef("info message")
		if n == 0 {
			t.Errorf("Expected WriteLinef to write bytes at info level, got 0")
		}
	})

	t.Run("backward compat: CreateMainLogger debug=true => debug level", func(t *testing.T) {
		l := CreateMainLogger("T", 1, "%s", true, false)
		n, _ := l.Debugf("debug msg")
		if n == 0 {
			t.Errorf("Expected Debugf output when debug=true, got 0 bytes")
		}
	})

	t.Run("backward compat: CreateMainLogger debug=false => info level", func(t *testing.T) {
		l := CreateMainLogger("T", 1, "%s", false, false)
		n, _ := l.Debugf("debug msg")
		if n != 0 {
			t.Errorf("Expected Debugf suppressed when debug=false, got %d bytes", n)
		}
		n, _ = l.WriteLinef("info msg")
		if n == 0 {
			t.Errorf("Expected WriteLinef output at default level, got 0 bytes")
		}
	})
}

// captureLogger is a stub used by TestLoggerLevelFiltering to avoid real stdout.
type captureLogger struct{ capture *[]string }

func (c *captureLogger) WriteLinef(f string, a ...interface{}) (int, error) { return 0, nil }
func (c *captureLogger) Debugf(f string, a ...interface{}) (int, error)     { return 0, nil }
func (c *captureLogger) WriteLine(line string) (int, error)                 { return 0, nil }
func (c *captureLogger) Debug(line string) (int, error)                     { return 0, nil }
func (c *captureLogger) Write(p []byte) (int, error)                        { return 0, nil }
