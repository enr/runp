package core

import (
	"fmt"

	"github.com/hashicorp/go-version"
)

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
			ui.Debugf(`Runp version precondition satisfied: %s %s %s`, currentVersion, p.Operator, targetVersion)
			return PreconditionVerifyResult{Vote: Proceed}
		}
	case LessThanOrEqual:
		if currentVersion.LessThanOrEqual(targetVersion) {
			ui.Debugf(`Runp version precondition satisfied: %s %s %s`, currentVersion, p.Operator, targetVersion)
			return PreconditionVerifyResult{Vote: Proceed}
		}
	case Equal:
		if currentVersion.Equal(targetVersion) {
			ui.Debugf(`Runp version precondition satisfied: %s %s %s`, currentVersion, p.Operator, targetVersion)
			return PreconditionVerifyResult{Vote: Proceed}
		}
	case GreaterThanOrEqual:
		if currentVersion.GreaterThanOrEqual(targetVersion) {
			ui.Debugf(`Runp version precondition satisfied: %s %s %s`, currentVersion, p.Operator, targetVersion)
			return PreconditionVerifyResult{Vote: Proceed}
		}
	case GreaterThan:
		if currentVersion.GreaterThan(targetVersion) {
			ui.Debugf(`Runp version precondition satisfied: %v %v %v`, currentVersion, p.Operator, targetVersion)
			return PreconditionVerifyResult{Vote: Proceed}
		}
	}

	return PreconditionVerifyResult{
		Vote:    Stop,
		Reasons: []string{fmt.Sprintf(`version "%s" is not %s than %v`, currentVersion, p.Operator, targetVersion)},
	}
}

// IsSet ...
func (p *RunpVersionPrecondition) IsSet() bool {
	return p.Operator != "" && p.Version != ""
}
