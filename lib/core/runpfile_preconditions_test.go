package core

import (
	"runtime"
	"testing"
)

// go test -timeout 30s -run ^TestOsPreconditionHappyPath$ github.com/enr/runp/lib/core
func TestOsPreconditionHappyPath(t *testing.T) {

	sut := &OsPrecondition{Inclusions: []string{runtime.GOOS}}
	res := sut.Verify()
	if res.Vote != Proceed {
		t.Errorf("Expected Proceed but got %v", res.Vote)
	}
}

func TestRunpVersionPreconditionHappyPath(t *testing.T) {

	// current version
	Version = "1.0.0"

	sut := &RunpVersionPrecondition{
		Version:  "1.0.1",
		Operator: LessThan,
	}
	res := sut.Verify()
	if res.Vote != Proceed {
		t.Errorf("Expected Proceed but got %v: %v", res.Vote, res.Reasons)
	}
}
