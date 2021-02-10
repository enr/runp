package core

import (
	"io"
)

// RunpCommand is interface for all command types: host, container, aws process...
type RunpCommand interface {
	Pid() int
	Stdout(stdout io.Writer)
	Stderr(stderr io.Writer)
	Start() error
	Run() error
	Stop() error
	Wait() error
}
