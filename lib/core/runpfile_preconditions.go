package core

import (
	"fmt"
	"runtime"

	version "github.com/hashicorp/go-version"
)

// Precondition is.
type Precondition interface {
	Verify() error
}

// RunpVersionPrecondition checks Runp version.
type RunpVersionPrecondition struct {
}

func (p *RunpVersionPrecondition) Verify() error {
	current, err := version.NewVersion(Version)
	if err != nil {
		return err
	}
	required, err := version.NewVersion("1.5+metadata")
	if err != nil {
		return err
	}
	if !current.GreaterThanOrEqual(required) {
		return fmt.Errorf(`current "%s" but required is "%s" `, current, required)
	}
	return nil
}

func NewOsPrecondition(spec map[string]interface{}) (OsPrecondition, error) {
	var inclusion string
	current := runtime.GOOS
	if val, ok := spec["inclusion"]; ok {
		inclusion = fmt.Sprintf("%v", val)
	}
	return OsPrecondition{
		current:   current,
		inclusion: inclusion,
	}, nil
}

// OsPrecondition verify os.
type OsPrecondition struct {
	inclusion string
	current   string
}

func (p *OsPrecondition) Verify() error {
	if p.inclusion != "" && p.inclusion != p.current {
		return fmt.Errorf(`inclusion "%s" but current is "%s" `, p.inclusion, p.current)
	}
	return nil
}
