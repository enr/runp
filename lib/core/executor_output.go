package core

import (
	"bufio"
	"os"
)

func (e *RunpfileExecutor) readProcessOutput(r *os.File, process RunpProcess, logger Logger) {
	readProcessOutputInternal(r, process, logger, nil)
}

// readProcessOutputToChannel reads lines from r, writes each to logger, and
// forwards them to lineCh with a non-blocking send. It closes lineCh when the
// process output stream ends (EOF or error).
func (e *RunpfileExecutor) readProcessOutputToChannel(r *os.File, process RunpProcess, logger Logger, lineCh chan<- string) {
	readProcessOutputInternal(r, process, logger, lineCh)
}

func readProcessOutputInternal(r *os.File, process RunpProcess, logger Logger, lineCh chan<- string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
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
	if lineCh != nil {
		close(lineCh)
	}
	if err := scanner.Err(); err != nil {
		logger.WriteLinef("Failed to read output from process %s: %v", process.ID(), err)
	}
}
