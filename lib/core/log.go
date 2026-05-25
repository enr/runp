package core

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"sync"

	ct "github.com/daviddengcn/go-colortext"
)

// destructiveAnsiRe matches ANSI/VT sequences that modify screen state rather
// than text attributes: erase-display, absolute cursor positioning, private-mode
// toggles (alternate buffer, cursor visibility), DEC save/restore, and OSC
// sequences (window title, etc.). Color and attribute sequences are left intact.
var destructiveAnsiRe = regexp.MustCompile(
	`\x1b\[[\d;]*[JH]` + // erase display (J) / absolute cursor position (H)
		`|\x1b\[\?[\d;]*[hl]` + // private modes: alternate screen, cursor visibility, …
		`|\x1b[78]` + // DEC save-cursor (ESC 7) / restore-cursor (ESC 8)
		`|\x1b\][^\x07\x1b]*\x07`, // OSC: window title, icon name, …
)

// sanitizeLine removes content that would corrupt the terminal when forwarded
// from a child process: destructive ANSI sequences and bare carriage returns.
func sanitizeLine(s string) string {
	s = destructiveAnsiRe.ReplaceAllString(s, "")
	return strings.ReplaceAll(s, "\r", "")
}

// LoggerConfig contains configuration for a logger.
type LoggerConfig struct {
	// Level sets the minimum log level. Zero value (LogLevelUnset) is treated
	// as LogLevelInfo unless Debug is also set.
	Level LogLevel
	// Debug is a legacy field. If Level is LogLevelUnset and Debug is true,
	// the effective level is LogLevelDebug.
	Debug bool
	Color bool
}

// Logger writes out messages from the main program and output from the running processes.
type Logger interface {
	WriteLinef(format string, a ...interface{}) (int, error)
	Debugf(format string, a ...interface{}) (int, error)
	WriteLine(line string) (int, error)
	Debug(line string) (int, error)
	Write(p []byte) (int, error)
}

type clogger struct {
	idx     int
	bold    bool
	proc    string
	longest int
	format  string
	level   LogLevel
	colors  bool
}

// process log color index
var ci int
var mutex = new(sync.Mutex)

func (l *clogger) effectiveLevel() LogLevel {
	if l.level != LogLevelUnset {
		return l.level
	}
	return LogLevelInfo
}

// Debugf writes a debug-level message. Suppressed below LogLevelDebug.
func (l *clogger) Debugf(format string, a ...interface{}) (int, error) {
	return l.Debug(fmt.Sprintf(format, a...))
}

// WriteLinef writes an info-level message. Suppressed below LogLevelInfo.
func (l *clogger) WriteLinef(format string, a ...interface{}) (int, error) {
	return l.WriteLine(fmt.Sprintf(format, a...))
}

// Debug writes a debug-level line. Suppressed below LogLevelDebug.
func (l *clogger) Debug(line string) (int, error) {
	if l.effectiveLevel() >= LogLevelDebug {
		return l.doWriteLine(line)
	}
	return 0, nil
}

// WriteLine writes an info-level line. Suppressed below LogLevelInfo.
func (l *clogger) WriteLine(line string) (int, error) {
	if l.effectiveLevel() >= LogLevelInfo {
		return l.doWriteLine(line)
	}
	return 0, nil
}

func (l *clogger) doWriteLine(line string) (int, error) {
	if len(line) == 0 {
		return 0, nil
	}
	sanitized := sanitizeLine(line)
	if len(sanitized) == 0 {
		return 0, nil
	}
	mutex.Lock()
	if l.colors {
		ct.ResetColor()
		ct.ChangeColor(labelColors[l.idx].foreground, l.bold, labelColors[l.idx].background, false)
	}
	fmt.Printf(l.format, l.proc)
	if l.colors {
		ct.ResetColor()
	}
	fmt.Print(sanitized)
	if sanitized[len(sanitized)-1] != '\n' {
		fmt.Print("\r\n")
	}
	mutex.Unlock()
	return len(line), nil
}

// Write pipes raw process output. Always printed regardless of log level.
func (l *clogger) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	buf := bytes.NewBuffer(p)
	wrote := 0
	for {
		line, err := buf.ReadBytes('\n')
		if len(line) > 1 {
			s := sanitizeLine(string(line))
			// Blank lines (and lines that reduce to empty after sanitization) are
			// dropped: in a multiplexed stream a label-less blank line is ambiguous.
			if s == "" || s == "\n" {
				wrote += len(line)
			} else {
				mutex.Lock()
				if l.colors {
					ct.ResetColor()
				}
				if l.colors {
					ct.ChangeColor(labelColors[l.idx].foreground, l.bold, labelColors[l.idx].background, false)
				}
				fmt.Printf(l.format, l.proc)
				if l.colors {
					ct.ResetColor()
				}
				fmt.Print(s)
				if s[len(s)-1] != '\n' {
					fmt.Print("\r\n")
				}
				mutex.Unlock()
				wrote += len(line)
			}
		}
		if err != nil {
			break
		}
	}
	return len(p), nil
}

// createProcessLogger creates a logger for a single process's output stream.
func createProcessLogger(proc string, longest int, cfg LoggerConfig) Logger {
	level := cfg.Level
	if level == LogLevelUnset {
		if cfg.Debug {
			level = LogLevelDebug
		} else {
			level = LogLevelInfo
		}
	}
	return CreateMainLoggerWithLevel(proc, longest, fmt.Sprintf("%%%ds | ", longest), level, cfg.Color)
}

// CreateMainLogger creates a logger instance. debug=true is equivalent to
// LogLevelDebug; use CreateMainLoggerWithLevel for full level control.
func CreateMainLogger(proc string, longest int, format string, debug bool, colorize bool) Logger {
	level := LogLevelInfo
	if debug {
		level = LogLevelDebug
	}
	return CreateMainLoggerWithLevel(proc, longest, format, level, colorize)
}

// CreateMainLoggerWithLevel creates a logger instance with an explicit level.
func CreateMainLoggerWithLevel(proc string, longest int, format string, level LogLevel, colorize bool) Logger {
	n := ` `
	if proc != "" {
		n = proc
	}
	f := "\r" + format
	mutex.Lock()
	idx := ci % len(labelColors)
	bold := (ci/len(labelColors))%2 == 1
	ci++
	mutex.Unlock()
	return &clogger{idx: idx, bold: bold, proc: n, longest: longest, format: f, level: level, colors: colorize}
}

// ResetColor resets the foreground and background to original colors
func ResetColor() {
	ct.ResetColor()
}
