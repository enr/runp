package core

import (
	"runtime"
	"testing"
)

// go test -timeout 30s -run ^TestOsPreconditionInclusionOk$ github.com/enr/runp/lib/core
func TestOsPreconditionInclusionOk(t *testing.T) {

	spec := map[string]interface{}{
		"inclusion": runtime.GOOS,
	}

	sut, err := NewOsPrecondition(spec)
	if err != nil {
		t.Errorf("Error %v", err)
	}
	err = sut.Verify()
	if err != nil {
		t.Errorf("Error %v", err)
	}
}
