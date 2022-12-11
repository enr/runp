package core

import (
	"fmt"
	"runtime"
)

// Precondition is.
type Precondition interface {
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
	Os OsPrecondition
}

// Verify ...
func (p *Preconditions) Verify() PreconditionVerifyResult {
	preconditions := []Precondition{
		&p.Os,
	}
	var vr PreconditionVerifyResult
	res := PreconditionVerifyResult{Vote: Proceed, Reasons: []string{}}
	for _, v := range preconditions {
		ui.WriteLinef("in preconditions verify %v \n", v)
		vr = v.Verify()
		ui.WriteLinef("in preconditions verify %v \n", vr)
		if vr.Vote == Proceed {
			continue
		}
		res.Vote = Stop
		res.Reasons = append(res.Reasons, vr.Reasons...)
	}
	return res
}

// // RunpVersionPrecondition checks Runp version.
// type RunpVersionPrecondition struct {
// }

// func (p *RunpVersionPrecondition) Verify() error {
// 	current, err := version.NewVersion(Version)
// 	if err != nil {
// 		return err
// 	}
// 	required, err := version.NewVersion("1.5+metadata")
// 	if err != nil {
// 		return err
// 	}
// 	if !current.GreaterThanOrEqual(required) {
// 		return fmt.Errorf(`current "%s" but required is "%s" `, current, required)
// 	}
// 	return nil
// }

// func NewOsPrecondition(spec map[string]interface{}) (OsPrecondition, error) {
// 	var inclusion string
//
// 	if val, ok := spec["inclusion"]; ok {
// 		inclusion = fmt.Sprintf("%v", val)
// 	}
// 	return OsPrecondition{
// 		current:   current,
// 		inclusion: inclusion,
// 	}, nil
// }

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
	// if p.inclusion != "" && p.inclusion != p.current {
	// 	return fmt.Errorf(`inclusion "%s" but current is "%s" `, p.inclusion, p.current)
	// }
	return PreconditionVerifyResult{
		Vote:    Stop,
		Reasons: []string{fmt.Sprintf(`current os "%s" not in %v`, current, p.Inclusions)},
	}
}
