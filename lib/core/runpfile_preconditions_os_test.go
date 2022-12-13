package core

import (
	"runtime"
	"testing"
)

type osPreconditionTestCase struct {
	current    string
	inclusions []string
	vote       PreconditionVote
}

var osTestCases = []osPreconditionTestCase{
	{current: "windows", inclusions: []string{"windows"}, vote: Proceed},
	{current: "windows", inclusions: []string{"linux", "darwin"}, vote: Stop},
	{current: "windows", inclusions: []string{}, vote: Stop},
}

func TestOsPrecondition(t *testing.T) {

	ui = CreateMainLogger(" ", 6, "%s> ", true, false)
	for _, c := range osTestCases {
		sut := &OsPrecondition{
			Inclusions: c.inclusions,
			osProvider: func() string {
				return c.current
			},
		}
		res := sut.Verify()
		if res.Vote != c.vote {
			t.Errorf("Expected %v but got %v for %s", c.vote, res.Vote, c.current)
		}

	}
}

// go test -timeout 30s -run ^TestOsPreconditionHappyPath$ github.com/enr/runp/lib/core
func TestOsPreconditionHappyPath(t *testing.T) {

	sut := &OsPrecondition{Inclusions: []string{runtime.GOOS}}
	res := sut.Verify()
	if res.Vote != Proceed {
		t.Errorf("Expected Proceed but got %v", res.Vote)
	}
}
