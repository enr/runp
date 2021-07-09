package core

import (
	"bytes"
	"fmt"
	"sync"

	ct "github.com/daviddengcn/go-colortext"
)

// LoggerConfig contains configuration for a logger.
type LoggerConfig struct {
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
	proc    string
	longest int
	format  string
	debug   bool
	colors  bool
}

// process log color index
var ci int
var mutex = new(sync.Mutex)

func (l *clogger) Debugf(format string, a ...interface{}) (int, error) {
	if l.debug {
		return l.WriteLine(fmt.Sprintf(format, a...))
	}
	return 0, nil
}

func (l *clogger) WriteLinef(format string, a ...interface{}) (int, error) {
	return l.WriteLine(fmt.Sprintf(format, a...))
}

func (l *clogger) Debug(line string) (int, error) {
	if l.debug {
		return l.WriteLine(line)
	}
	return 0, nil
}

func (l *clogger) WriteLine(line string) (int, error) {
	if len(line) == 0 {
		return 0, nil
	}
	mutex.Lock()
	if l.colors {
		ct.ChangeColor(labelColors[l.idx].foreground, false, labelColors[l.idx].background, false)
	}
	fmt.Printf(l.format, l.proc)
	if l.colors {
		ct.ResetColor()
	}
	fmt.Print(line)
	if line[len(line)-1] != '\n' {
		fmt.Println()
	}
	mutex.Unlock()
	return len(line), nil
}

func (l *clogger) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	buf := bytes.NewBuffer(p)
	wrote := 0
	for {
		line, err := buf.ReadBytes('\n')
		if len(line) > 1 {
			s := string(line)

			mutex.Lock()
			if l.colors {
				ct.ChangeColor(labelColors[l.idx].foreground, false, labelColors[l.idx].background, false)
			}
			fmt.Printf(l.format, l.proc)
			if l.colors {
				ct.ResetColor()
			}
			fmt.Print(s)
			mutex.Unlock()

			wrote += len(line)
		}
		if err != nil {
			break
		}
	}
	if len(p) > 0 && p[len(p)-1] != '\n' {
		fmt.Println()
	}
	return len(p), nil
}

// create logger instance for processes output.
func createProcessLogger(proc string, longest int, processLoggerConfiguration LoggerConfig) Logger {
	return CreateMainLogger(proc, longest, fmt.Sprintf("%%%ds | ", longest), processLoggerConfiguration.Debug, processLoggerConfiguration.Color)
}

// CreateMainLogger creates logger instance for main process (runp).
func CreateMainLogger(proc string, longest int, format string, debug bool, colorize bool) Logger {
	n := ` `
	if proc != "" {
		n = proc
	}
	l := &clogger{idx: ci, proc: n, longest: longest, format: format, debug: debug, colors: colorize}
	ci++
	if ci >= len(labelColors) {
		ci = 0
	}
	return l
}
