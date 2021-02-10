package qac

import (
	"fmt"
	"strconv"
)

func (a *StatusAssertion) verify(context planContext) AssertionResult {
	result := AssertionResult{
		description: fmt.Sprintf(`status for %s`, context.commandResult.execution),
	}
	commandErrorIsAcceptable := false
	commandResult := context.commandResult
	if a.GreaterThan != "" {
		i, err := strconv.Atoi(a.GreaterThan)
		if err != nil {
			result.addError(err)
			// block
			return result
		}
		if commandResult.exitCode <= i {
			result.addErrorf(`exit code expected GT %d got %d`, i, commandResult.exitCode)
		}
		commandErrorIsAcceptable = true
	}
	if a.LesserThan != "" {
		i, err := strconv.Atoi(a.LesserThan)
		if err != nil {
			result.addError(err)
			// block
			return result
		}
		if commandResult.exitCode >= i {
			result.addErrorf(`exit code expected LT %d got %d`, i, commandResult.exitCode)
		}
		commandErrorIsAcceptable = true
	}
	if a.EqualsTo != "" {
		i, err := strconv.Atoi(a.EqualsTo)
		if err != nil {
			result.addError(err)
			// block
			return result
		}
		if commandResult.exitCode != i {
			result.addErrorf(`exit code expected EQUALS %d got %d`, i, commandResult.exitCode)
		}
		commandErrorIsAcceptable = i > 0
	}
	if commandResult.err != nil && !commandErrorIsAcceptable {
		result.addErrorf(`command execution error: %v`, commandResult.err)
	}
	return result
}
