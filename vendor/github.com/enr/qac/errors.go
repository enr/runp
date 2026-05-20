package qac

// ErrorKind categorises an Error so consumers can distinguish failure modes.
type ErrorKind int

const (
	// KindAssertionFailure means an assertion's condition was not met.
	KindAssertionFailure ErrorKind = iota
	// KindInfrastructure means a system-level operation failed (file I/O, path resolution).
	KindInfrastructure
	// KindConfiguration means the test plan itself is invalid (YAML, unknown fields, wrong field usage).
	KindConfiguration
)

// Error is a structured error that carries a kind for programmatic classification.
// It wraps the underlying cause so errors.As / errors.Is work through the chain.
type Error struct {
	Kind  ErrorKind
	Cause error
	msg   string
}

func (e *Error) Error() string { return e.msg }
func (e *Error) Unwrap() error { return e.Cause }

// asInfraError wraps an existing error as KindInfrastructure, preserving its message and chain.
func asInfraError(err error) *Error {
	return &Error{Kind: KindInfrastructure, Cause: err, msg: err.Error()}
}

// asConfigError wraps an existing error as KindConfiguration, preserving its message and chain.
func asConfigError(err error) *Error {
	return &Error{Kind: KindConfiguration, Cause: err, msg: err.Error()}
}
