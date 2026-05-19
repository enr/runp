package core

import (
	"fmt"
	"sort"

	yaml "gopkg.in/yaml.v3"
)

// ValidationResult holds errors and warnings produced by ValidateRunpfile.
type ValidationResult struct {
	Errors   []error
	Warnings []string
}

// Valid returns true when there are no validation errors.
func (r *ValidationResult) Valid() bool {
	return len(r.Errors) == 0
}

// ValidateRunpfile runs all static checks on a loaded Runpfile and returns
// a ValidationResult. It does not start any process.
//
// Errors (block execution):
//   - no units defined
//   - unit with multiple or missing process types
//   - variable references that are not declared in the vars section
//   - circular variable references in the vars section
//
// Warnings (informational, do not fail validation):
//   - OS or runp-version preconditions that would cause a unit to be skipped
//     on the current machine
func ValidateRunpfile(rf *Runpfile) ValidationResult {
	result := ValidationResult{}

	_, structErrs := IsRunpfileValid(rf)
	result.Errors = append(result.Errors, structErrs...)

	result.Errors = append(result.Errors, validateVarExpansion(rf)...)
	result.Errors = append(result.Errors, validateVariableRefs(rf)...)

	result.Errors = append(result.Errors, validateDependsOn(rf)...)

	result.Warnings = append(result.Warnings, collectPreconditionWarnings(rf)...)

	return result
}

// validateVarExpansion checks that var values do not reference undeclared vars
// and contain no circular references (vars-in-vars).
func validateVarExpansion(rf *Runpfile) []error {
	if len(rf.Vars) == 0 {
		return nil
	}
	if _, err := ExpandVars(rf.Vars); err != nil {
		return []error{fmt.Errorf("vars section: %w", err)}
	}
	return nil
}

// validateVariableRefs checks that every {{vars NAME}} reference inside a
// unit resolves to a name declared in the Runpfile vars section.
func validateVariableRefs(rf *Runpfile) []error {
	var errs []error

	// Collect unit names in deterministic order for stable error messages.
	unitNames := make([]string, 0, len(rf.Units))
	for name := range rf.Units {
		unitNames = append(unitNames, name)
	}
	sort.Strings(unitNames)

	for _, unitName := range unitNames {
		unit := rf.Units[unitName]
		data, err := yaml.Marshal(unit)
		if err != nil {
			continue
		}
		refs := extractVarRefs(string(data))
		for _, ref := range refs {
			if _, declared := rf.Vars[ref]; !declared {
				errs = append(errs, fmt.Errorf(
					"unit %q: variable %q referenced but not declared in vars section",
					unitName, ref,
				))
			}
		}
	}
	return errs
}

// extractVarRefs returns the distinct variable names referenced via
// {{vars NAME}} in s, in order of first appearance.
func extractVarRefs(s string) []string {
	matches := varsRegexp.FindAllStringSubmatch(s, -1)
	seen := map[string]bool{}
	var refs []string
	for _, m := range matches {
		name := m[1]
		if !seen[name] {
			seen[name] = true
			refs = append(refs, name)
		}
	}
	return refs
}

// collectPreconditionWarnings returns warning strings for OS and runp-version
// preconditions that would cause units (or the whole Runpfile) to be skipped
// on the current machine. These are informational: a Runpfile designed for
// another OS is still structurally valid.
func collectPreconditionWarnings(rf *Runpfile) []string {
	var warnings []string

	warnings = append(warnings, preconditionWarnings("", &rf.Preconditions)...)

	unitNames := make([]string, 0, len(rf.Units))
	for name := range rf.Units {
		unitNames = append(unitNames, name)
	}
	sort.Strings(unitNames)

	for _, unitName := range unitNames {
		unit := rf.Units[unitName]
		warnings = append(warnings, preconditionWarnings(unitName, &unit.Preconditions)...)
	}
	return warnings
}

func preconditionWarnings(scope string, p *Preconditions) []string {
	label := "global"
	if scope != "" {
		label = fmt.Sprintf("unit %q", scope)
	}
	var warnings []string
	for _, pc := range []Precondition{&p.Os, &p.Runp} {
		if !pc.IsSet() {
			continue
		}
		if pr := pc.Verify(); pr.Vote == Stop {
			for _, reason := range pr.Reasons {
				warnings = append(warnings, fmt.Sprintf("%s: %s (unit will be skipped at runtime)", label, reason))
			}
		}
	}
	return warnings
}
