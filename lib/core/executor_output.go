package core

import (
	"bufio"
	"os"
)

func (e *RunpfileExecutor) readProcessOutput(r *os.File, process RunpProcess, logger Logger) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		logger.Write(scanner.Bytes())
	}
	if err := scanner.Err(); err != nil {
		logger.WriteLinef("Failed to read output from process %s: %v", process.ID(), err)
	}
}
