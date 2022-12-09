package core

// Precondition is.
type Precondition interface {
	Verify() error
}

// RunpVersionPrecondition checks Runp version.
type RunpVersionPrecondition struct {
}

func (p *RunpVersionPrecondition) Verify() error {
	return nil
}

// OsPrecondition verify os.
type OsPrecondition struct {
}

func (p *OsPrecondition) Verify() error {
	return nil
}
