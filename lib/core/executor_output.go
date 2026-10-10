package core

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
)

// maxOutputLineSize is the length after which a line of process output is
// split: a longer line is logged in chunks of this size instead of being
// buffered without limit.
const maxOutputLineSize = 1 << 20 // 1 MiB

func (e *RunpfileExecutor) readProcessOutput(r *os.File, process RunpProcess, logger Logger) {
	readProcessOutputInternal(r, process, logger, nil)
}

// readProcessOutputToChannel reads lines from r, writes each to logger, and
// forwards them to lineCh with a non-blocking send. It closes lineCh when the
// process output stream ends (EOF or error).
func (e *RunpfileExecutor) readProcessOutputToChannel(r *os.File, process RunpProcess, logger Logger, lineCh chan<- string) {
	readProcessOutputInternal(r, process, logger, lineCh)
}

// readProcessOutputInternal reads r until EOF. It never stops reading early
// (e.g. on a very long line): a process whose output is no longer read blocks
// as soon as the pipe buffer is full.
func readProcessOutputInternal(r io.Reader, process RunpProcess, logger Logger, lineCh chan<- string) {
	emit := func(b []byte) {
		line := string(bytes.TrimSuffix(bytes.TrimSuffix(b, []byte("\n")), []byte("\r")))
		logger.Write([]byte(line + "\n"))
		if lineCh != nil {
			// Non-blocking: drop lines if nobody is reading (e.g. after readiness
			// is already signalled). The logger above still receives every line.
			select {
			case lineCh <- line:
			default:
			}
		}
	}
	reader := bufio.NewReader(r)
	var buf []byte
	var readErr error
	for {
		chunk, err := reader.ReadSlice('\n')
		buf = append(buf, chunk...)
		if errors.Is(err, bufio.ErrBufferFull) {
			if len(buf) >= maxOutputLineSize {
				emit(buf)
				buf = buf[:0]
			}
			continue
		}
		if len(buf) > 0 {
			emit(buf)
			buf = buf[:0]
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				readErr = err
			}
			break
		}
	}
	if lineCh != nil {
		close(lineCh)
	}
	if readErr != nil {
		logger.WriteLinef("Failed to read output from process %s: %v", process.ID(), readErr)
		// Keep draining so the process never blocks writing to a full pipe.
		io.Copy(io.Discard, r)
	}
}
