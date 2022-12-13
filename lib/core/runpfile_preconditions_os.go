package core

import (
	"fmt"
	"runtime"
)

var defaultOsProvider = func() string {
	return runtime.GOOS
}

// OsPrecondition verify os.
type OsPrecondition struct {
	// linux | darwin | windows
	Inclusions []string

	osProvider func() string
}

// Verify ...
func (p *OsPrecondition) Verify() PreconditionVerifyResult {
	if p.osProvider == nil {
		p.osProvider = defaultOsProvider
	}
	current := p.osProvider()
	for _, v := range p.Inclusions {
		if v == current {
			return PreconditionVerifyResult{Vote: Proceed}
		}
	}
	return PreconditionVerifyResult{
		Vote:    Stop,
		Reasons: []string{fmt.Sprintf(`current os "%s" not in %v`, current, p.Inclusions)},
	}
}

// IsSet ...
func (p *OsPrecondition) IsSet() bool {
	return len(p.Inclusions) > 0
}
