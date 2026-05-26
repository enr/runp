package qac

import (
	"fmt"
	"os"
)

// specSkipReason returns a non-empty human-readable reason when the spec should
// be skipped, or an empty string when it should run normally.
//
// Evaluation order:
//  1. skip: true  — unconditional static skip.
//  2. skip_if.env_set — skip when the named variable is defined (any value).
//  3. skip_if.env_value — skip when any variable equals its specified value.
func specSkipReason(spec Spec) string {
	if spec.Skip {
		return "skip: true"
	}
	si := spec.SkipIf
	if si.EnvSet != "" {
		if _, defined := os.LookupEnv(si.EnvSet); defined {
			return fmt.Sprintf("env %s is set", si.EnvSet)
		}
	}
	for k, v := range si.EnvValue {
		if os.Getenv(k) == v {
			return fmt.Sprintf("%s=%s", k, v)
		}
	}
	return ""
}
