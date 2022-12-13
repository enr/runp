package core

import (
	"testing"
)

type runpPreconditionTestCase struct {
	current  string
	value    string
	operator VersionComparationOperator
	vote     PreconditionVote
}

var testCases = []runpPreconditionTestCase{
	{current: "1.0.0", value: "1.0.0-RC1", operator: LessThan, vote: Stop},
	{current: "1.0.0", value: "1.0.0-RC1", operator: LessThanOrEqual, vote: Stop},
	{current: "1.0.0", value: "1.0.0-RC1", operator: GreaterThan, vote: Proceed},
	{current: "0.0.1", value: "0.0.2-RC1", operator: GreaterThan, vote: Stop},
	{current: "0.0.1", value: "0.0.1", operator: Equal, vote: Proceed},
}

func TestRunpVersionPrecondition(t *testing.T) {

	ui = CreateMainLogger(" ", 6, "%s> ", true, false)
	for _, c := range testCases {
		// current version
		Version = c.current

		sut := &RunpVersionPrecondition{
			Version:  c.value,
			Operator: c.operator,
		}
		res := sut.Verify()
		if res.Vote != c.vote {
			t.Errorf("Expected %v but got %v: %v", c.vote, res.Vote, res.Reasons)
		}

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
