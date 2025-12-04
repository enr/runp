package core

import (
	"testing"
)

func TestPreconditions(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})
	runpfilePath := "../../testdata/runpfiles/preconditions.yml"
	rp, err := LoadRunpfileFromPath(runpfilePath)
	if err != nil {
		t.Errorf("Runppfile %s, load error %v", runpfilePath, err)
	}
	isActuallyValid, _ := IsRunpfileValid(rp)
	if !isActuallyValid {
		t.Errorf("Expected runpfile valid but it is not\n")
	}
	units := rp.Units
	if len(units) != 1 {
		t.Errorf("Expected units #%d but got #%d\n", 1, len(units))
	}
}

func TestPreconditionVoteString(t *testing.T) {
	tests := []struct {
		name     string
		vote     PreconditionVote
		expected string
	}{
		{"Unknown", Unknown, "UNKNOWN"},
		{"Stop", Stop, "STOP"},
		{"Proceed", Proceed, "PROCEED"},
		{"Invalid", PreconditionVote(99), "99"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.vote.String()
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
