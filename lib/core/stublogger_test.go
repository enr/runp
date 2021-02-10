// +build darwin freebsd linux netbsd openbsd

package core

import (
	"bytes"
	"fmt"
)

var (
	testLogger = &stubLogger{}
)

type stubLogger struct {
	output []string
}

func (l *stubLogger) Debugf(format string, a ...interface{}) (int, error) {

	return l.WriteLine(fmt.Sprintf(format, a...))
}
func (l *stubLogger) WriteLinef(format string, a ...interface{}) (int, error) {
	return l.WriteLine(fmt.Sprintf(format, a...))
}
func (l *stubLogger) Debug(line string) (int, error) {

	return l.WriteLine(line)

}

func (l *stubLogger) WriteLine(line string) (int, error) {
	if len(line) == 0 {
		return 0, nil
	}
	mutex.Lock()
	l.output = append(l.output, line)
	mutex.Unlock()
	return len(line), nil
}

// write handler of logger.
func (l *stubLogger) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	buf := bytes.NewBuffer(p)
	for {
		line, err := buf.ReadBytes('\n')
		if len(line) > 1 {
			mutex.Lock()
			s := string(line)
			l.output = append(l.output, s)
			mutex.Unlock()
		}
		if err != nil {
			break
		}
	}
	return len(p), nil
}

func (l *stubLogger) outputLines() []string {
	return l.output
}

func createStubLogger(proc string, longest int, lc LoggerConfig) Logger {
	return testLogger
}
