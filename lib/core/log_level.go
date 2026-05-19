package core

import (
	"fmt"
	"strings"
)

// LogLevel controls how much output the logger produces.
// Higher values are more verbose.
type LogLevel int

const (
	// LogLevelUnset is the zero value; callers treat it as LogLevelInfo.
	LogLevelUnset LogLevel = 0
	// LogLevelError logs only error messages.
	LogLevelError LogLevel = 1
	// LogLevelWarn logs warnings and errors.
	LogLevelWarn LogLevel = 2
	// LogLevelInfo logs informational messages, warnings, and errors.
	LogLevelInfo LogLevel = 3
	// LogLevelDebug logs debug output and everything above.
	LogLevelDebug LogLevel = 4
	// LogLevelTrace logs the most verbose output.
	LogLevelTrace LogLevel = 5
)

// ParseLogLevel converts a string name to a LogLevel.
// Valid values (case-insensitive): error, warn, info, debug, trace.
func ParseLogLevel(s string) (LogLevel, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "error":
		return LogLevelError, nil
	case "warn", "warning":
		return LogLevelWarn, nil
	case "info":
		return LogLevelInfo, nil
	case "debug":
		return LogLevelDebug, nil
	case "trace":
		return LogLevelTrace, nil
	}
	return LogLevelUnset, fmt.Errorf("unknown log level %q; valid values: trace, debug, info, warn, error", s)
}

func (l LogLevel) String() string {
	switch l {
	case LogLevelError:
		return "error"
	case LogLevelWarn:
		return "warn"
	case LogLevelInfo:
		return "info"
	case LogLevelDebug:
		return "debug"
	case LogLevelTrace:
		return "trace"
	}
	return "info"
}
