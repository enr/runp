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
	result.Warnings = append(result.Warnings, collectAwaitWarnings(rf)...)

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

// implicitVars are set by runp itself when units start, so they can be
// referenced without being declared in the vars section.
var implicitVars = map[string]bool{
	"runp_root":           true,
	"runp_workdir":        true,
	"runp_file_separator": true,
}

// validateVariableRefs checks that every {{vars NAME}} reference inside a
// unit resolves to a name declared in the Runpfile vars section (or to an
// implicit var).
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
			if _, declared := rf.Vars[ref]; !declared && !implicitVars[ref] {
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

// collectAwaitWarnings warns about await blocks without timeout: they are
// ignored, the unit starts without waiting for the resource.
func collectAwaitWarnings(rf *Runpfile) []string {
	unitNames := make([]string, 0, len(rf.Units))
	for name := range rf.Units {
		unitNames = append(unitNames, name)
	}
	sort.Strings(unitNames)
	var warnings []string
	for _, name := range unitNames {
		unit := rf.Units[name]
		var await AwaitCondition
		switch {
		case unit.Host != nil:
			await = unit.Host.Await
		case unit.Container != nil:
			await = unit.Container.Await
		case unit.SSHTunnel != nil:
			await = unit.SSHTunnel.Await
		}
		if await.Resource != "" && await.Timeout == "" {
			warnings = append(warnings, fmt.Sprintf("unit %q: await.resource is set without await.timeout, the unit will not wait for %s", name, await.Resource))
		}
	}
	return warnings
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
