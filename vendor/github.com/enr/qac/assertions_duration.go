package qac

import (
	"fmt"
	"time"
)

func (a *DurationAssertion) verify(context planContext) AssertionResult {
	result := AssertionResult{
		description: fmt.Sprintf("duration for %s", context.commandResult.execution),
	}
	d := context.commandResult.duration

	if a.Max != "" {
		max, err := time.ParseDuration(a.Max)
		if err != nil {
			result.addConfigError(fmt.Errorf("invalid duration in max %q: %w", a.Max, err))
		} else if d > max {
			result.addErrorf("duration %s exceeded max %s", d.Round(time.Millisecond), a.Max)
		}
	}

	if a.Min != "" {
		min, err := time.ParseDuration(a.Min)
		if err != nil {
			result.addConfigError(fmt.Errorf("invalid duration in min %q: %w", a.Min, err))
		} else if d < min {
			result.addErrorf("duration %s is below min %s", d.Round(time.Millisecond), a.Min)
		}
	}

	return result
}
