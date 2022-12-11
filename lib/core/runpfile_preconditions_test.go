package core

import (
	"runtime"
	"testing"
)

// go test -timeout 30s -run ^TestOsPreconditionInclusionOk$ github.com/enr/runp/lib/core
func TestOsPreconditionInclusionOk(t *testing.T) {

	sut := &OsPrecondition{Inclusions: []string{runtime.GOOS}}
	res := sut.Verify()
	if res.Vote != Proceed {
		t.Errorf("Expected Proceed but got %v", res.Vote)
	}
}
