package core

import "time"

// RunpProcess is wrapper for running process
type RunpProcess interface {
	ID() string
	VerifyPreconditions() PreconditionVerifyResult
	SetPreconditions(Preconditions)
	SetID(string)
	StartCommand() (RunpCommand, error)
	StopCommand() (RunpCommand, error)
	StopTimeout() time.Duration
	Dir() string
	SetDir(string)
	ShouldWait() bool
	AwaitResource() string
	AwaitTimeout() string
	IsStartable() (bool, error)
	// PreStart is called immediately before the process command is started.
	// Implementations use it for setup that must happen before execution begins.
	PreStart() error
	// OnStarted is called after the process has been successfully started.
	// The pid parameter is the OS process ID (0 if unavailable).
	OnStarted(pid int)
	// PostStop is called after the process command has exited.
	// Implementations use it for cleanup (e.g. removing a PID file).
	PostStop()
}

// StartPlan defines how and when start process.
type StartPlan struct {
	Await AwaitCondition
}

// AwaitCondition defines time to wait for a resource.
type AwaitCondition struct {
	Resource string
	Timeout  string
}
