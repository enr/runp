package qac

import (
	"fmt"
)

func (a *StatusAssertion) verify(context planContext) AssertionResult {
	result := AssertionResult{
		description: fmt.Sprintf(`status for %s`, context.commandResult.execution),
	}
	commandErrorIsAcceptable := false
	commandResult := context.commandResult
	if a.GreaterThan != nil {
		if commandResult.exitCode <= *a.GreaterThan {
			result.addErrorf(`exit code expected GT %d got %d`, *a.GreaterThan, commandResult.exitCode)
		}
		commandErrorIsAcceptable = true
	}
	if a.LesserThan != nil {
		if commandResult.exitCode >= *a.LesserThan {
			result.addErrorf(`exit code expected LT %d got %d`, *a.LesserThan, commandResult.exitCode)
		}
		commandErrorIsAcceptable = true
	}
	if a.EqualsTo != nil {
		if commandResult.exitCode != *a.EqualsTo {
			result.addErrorf(`exit code expected EQUALS %d got %d`, *a.EqualsTo, commandResult.exitCode)
		}
		commandErrorIsAcceptable = commandErrorIsAcceptable || *a.EqualsTo > 0
	}
	if commandResult.err != nil && !commandErrorIsAcceptable {
		result.addErrorf(`command execution error: %v`, commandResult.err)
	}
	return result
}
