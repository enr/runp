package core

import (
	"fmt"
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
	Os      OsPrecondition          `yaml:"os"`
	Runp    RunpVersionPrecondition `yaml:"runp"`
	Hosts   EtcHostsPrecondition    `yaml:"hosts"`
	EnvVars EnvVarsPrecondition     `yaml:"env_vars"`
}

// Verify ...
func (p *Preconditions) Verify() PreconditionVerifyResult {
	preconditions := []Precondition{
		&p.Os,
		&p.Runp,
		&p.Hosts,
		&p.EnvVars,
	}
	var vr PreconditionVerifyResult
	res := PreconditionVerifyResult{Vote: Proceed, Reasons: []string{}}
	for _, v := range preconditions {
		if !v.IsSet() {
			continue
		}
		vr = v.Verify()
		if vr.Vote == Proceed {
			continue
		}
		res.Vote = Stop
		res.Reasons = append(res.Reasons, vr.Reasons...)
	}
	return res
}
