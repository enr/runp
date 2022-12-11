package core

import (
	"fmt"
	"runtime"

	"github.com/hashicorp/go-version"
)

// Precondition is.
type Precondition interface {
	IsSet() bool
	Verify() PreconditionVerifyResult
}

// PreconditionVote ...
type PreconditionVote int64

const (
	// Unknown vote
	Unknown PreconditionVote = 0
	// Stop the process
	Stop = 1
	// Proceed the process can be ran
	Proceed = 2
)

func (e PreconditionVote) String() string {
	switch e {
	case Unknown:
		return "UNKNOWN"
	case Stop:
		return "STOP"
	case Proceed:
		return "PROCEED"
	default:
		return fmt.Sprintf("%d", int64(e))
	}
}

// PreconditionVerifyResult ...
type PreconditionVerifyResult struct {
	Vote    PreconditionVote
	Reasons []string
}

// Preconditions ...
type Preconditions struct {
	Os   OsPrecondition
	Runp RunpVersionPrecondition
}

// Verify ...
func (p *Preconditions) Verify() PreconditionVerifyResult {
	preconditions := []Precondition{
		&p.Os,
		&p.Runp,
	}
	var vr PreconditionVerifyResult
	res := PreconditionVerifyResult{Vote: Proceed, Reasons: []string{}}
	for _, v := range preconditions {
		if !v.IsSet() {
			continue
		}
		vr = v.Verify()
		ui.Debugf("Precondition %v: %v", v, vr)
		if vr.Vote == Proceed {
			continue
		}
		res.Vote = Stop
		res.Reasons = append(res.Reasons, vr.Reasons...)
	}
	return res
}

// VersionComparationOperator ...
type VersionComparationOperator string

const (
	// None ...
	None VersionComparationOperator = "None"
	// LessThan ...
	LessThan = "LessThan"
	// LessThanOrEqual ...
	LessThanOrEqual = "LessThanOrEqual"
	// Equal ...
	Equal = "Equal"
	// GreaterThanOrEqual ...
	GreaterThanOrEqual = "GreaterThanOrEqual"
	// GreaterThan ...
	GreaterThan = "GreaterThan"
)

// RunpVersionPrecondition checks Runp version.
type RunpVersionPrecondition struct {
	Operator VersionComparationOperator
	Version  string
}

// Verify ...
func (p *RunpVersionPrecondition) Verify() PreconditionVerifyResult {
	currentVersion, err := version.NewVersion(Version)

	if err != nil {
		return PreconditionVerifyResult{
			Vote:    Stop,
			Reasons: []string{fmt.Sprintf(`error getting current version %v`, err)},
		}
	}
	targetVersion, err := version.NewVersion(p.Version)

	if err != nil {
		return PreconditionVerifyResult{
			Vote:    Stop,
			Reasons: []string{fmt.Sprintf(`error parsing version %v`, err)},
		}
	}

	switch p.Operator {
	case LessThan:
		if currentVersion.LessThan(targetVersion) {
			ui.Debugf(`Runp version precondition satisfied: %s %s %s`, p.Version, p.Operator, targetVersion)
			return PreconditionVerifyResult{Vote: Proceed}
		}
	case LessThanOrEqual:
		if currentVersion.LessThanOrEqual(targetVersion) {
			ui.Debugf(`Runp version precondition satisfied: %s %s %s`, p.Version, p.Operator, targetVersion)
			return PreconditionVerifyResult{Vote: Proceed}
		}
	case Equal:
		if currentVersion.Equal(targetVersion) {
			ui.Debugf(`Runp version precondition satisfied: %s %s %s`, p.Version, p.Operator, targetVersion)
			return PreconditionVerifyResult{Vote: Proceed}
		}
	case GreaterThanOrEqual:
		if currentVersion.GreaterThanOrEqual(targetVersion) {
			ui.Debugf(`Runp version precondition satisfied: %s %s %s`, p.Version, p.Operator, targetVersion)
			return PreconditionVerifyResult{Vote: Proceed}
		}
	case GreaterThan:
		if currentVersion.GreaterThan(targetVersion) {
			ui.Debugf(`Runp version precondition satisfied: %s %s %s`, p.Version, p.Operator, targetVersion)
			return PreconditionVerifyResult{Vote: Proceed}
		}
	}

	return PreconditionVerifyResult{
		Vote:    Stop,
		Reasons: []string{fmt.Sprintf(`version "%s" is not %s current %v`, p.Version, p.Operator, targetVersion)},
	}
}

// IsSet ...
func (p *RunpVersionPrecondition) IsSet() bool {
	return p.Operator != "" && p.Version != ""
}

// OsPrecondition verify os.
type OsPrecondition struct {
	Inclusions []string
}

// Verify ...
func (p *OsPrecondition) Verify() PreconditionVerifyResult {
	current := runtime.GOOS
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
