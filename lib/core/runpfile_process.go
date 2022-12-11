package core

import "time"

// RunpProcess is.
// questo viene usato come running process
type RunpProcess interface {
	ID() string
	VerifyPreconditions() PreconditionVerifyResult
	SetPreconditions(Preconditions)
	SetID(string)
	StartCommand() (RunpCommand, error)
	StopCommand() RunpCommand
	StopTimeout() time.Duration
	Dir() string
	SetDir(string)
	ShouldWait() bool
	AwaitResource() string
	AwaitTimeout() string
	IsStartable() (bool, error)
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
