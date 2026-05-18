package core

import (
	"strings"
	"testing"
)

func TestValidateRunpfile(t *testing.T) {
	ui = CreateMainLogger(" ", 6, "TEST", false, false)

	t.Run("valid Runpfile returns no errors", func(t *testing.T) {
		rf := &Runpfile{
			Vars: map[string]string{"greeting": "hello"},
			Units: map[string]*RunpUnit{
				"my-unit": {
					Name: "my-unit",
					Host: &HostProcess{CommandLine: "echo {{vars greeting}}"},
				},
			},
		}
		result := ValidateRunpfile(rf)
		if !result.Valid() {
			t.Errorf("Expected valid Runpfile, got errors: %v", result.Errors)
		}
		if len(result.Warnings) != 0 {
			t.Errorf("Expected no warnings, got: %v", result.Warnings)
		}
	})

	t.Run("no units produces error", func(t *testing.T) {
		rf := &Runpfile{Units: map[string]*RunpUnit{}}
		result := ValidateRunpfile(rf)
		if result.Valid() {
			t.Error("Expected invalid result for Runpfile with no units")
		}
		if !containsErrorMsg(result.Errors, "No units defined") {
			t.Errorf("Expected 'No units defined' error, got: %v", result.Errors)
		}
	})

	t.Run("unit with no process type produces error", func(t *testing.T) {
		rf := &Runpfile{
			Units: map[string]*RunpUnit{
				"empty-unit": {Name: "empty-unit"},
			},
		}
		result := ValidateRunpfile(rf)
		if result.Valid() {
			t.Error("Expected invalid result for unit with no process type")
		}
		if !containsErrorMsg(result.Errors, "must define exactly one process type") {
			t.Errorf("Expected 'must define exactly one process type' error, got: %v", result.Errors)
		}
	})

	t.Run("unit with multiple process types produces error", func(t *testing.T) {
		rf := &Runpfile{
			Units: map[string]*RunpUnit{
				"multi-unit": {
					Name:      "multi-unit",
					Host:      &HostProcess{CommandLine: "echo hi"},
					Container: &ContainerProcess{Image: "alpine"},
				},
			},
		}
		result := ValidateRunpfile(rf)
		if result.Valid() {
			t.Error("Expected invalid result for unit with multiple process types")
		}
		if !containsErrorMsg(result.Errors, "mutually exclusive") {
			t.Errorf("Expected 'mutually exclusive' error, got: %v", result.Errors)
		}
	})

	t.Run("undefined variable reference produces error", func(t *testing.T) {
		rf := &Runpfile{
			Units: map[string]*RunpUnit{
				"my-unit": {
					Name: "my-unit",
					Host: &HostProcess{CommandLine: "echo {{vars missing_var}}"},
				},
			},
		}
		result := ValidateRunpfile(rf)
		if result.Valid() {
			t.Error("Expected invalid result for undefined variable reference")
		}
		if !containsErrorMsg(result.Errors, "missing_var") {
			t.Errorf("Expected error mentioning 'missing_var', got: %v", result.Errors)
		}
		if !containsErrorMsg(result.Errors, "not declared in vars section") {
			t.Errorf("Expected 'not declared in vars section' error, got: %v", result.Errors)
		}
	})

	t.Run("declared variable reference passes", func(t *testing.T) {
		rf := &Runpfile{
			Vars: map[string]string{"greeting": "hello"},
			Units: map[string]*RunpUnit{
				"my-unit": {
					Name: "my-unit",
					Host: &HostProcess{CommandLine: "echo {{vars greeting}}"},
				},
			},
		}
		result := ValidateRunpfile(rf)
		if !result.Valid() {
			t.Errorf("Expected valid result, got errors: %v", result.Errors)
		}
	})

	t.Run("OS precondition mismatch produces warning not error", func(t *testing.T) {
		rf := &Runpfile{
			Units: map[string]*RunpUnit{
				"my-unit": {
					Name: "my-unit",
					Host: &HostProcess{CommandLine: "echo hi"},
					Preconditions: Preconditions{
						Os: OsPrecondition{
							Inclusions: []string{"definitely-not-this-os"},
						},
					},
				},
			},
		}
		result := ValidateRunpfile(rf)
		if !result.Valid() {
			t.Errorf("Expected valid (OS preconditions are warnings), got errors: %v", result.Errors)
		}
		if len(result.Warnings) == 0 {
			t.Error("Expected at least one warning for OS precondition mismatch")
		}
		if !containsString(result.Warnings, "unit will be skipped") {
			t.Errorf("Expected warning to mention 'unit will be skipped', got: %v", result.Warnings)
		}
	})
}

func TestExtractVarRefs(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"echo {{vars greeting}}", []string{"greeting"}},
		{"{{vars a}} and {{vars b}}", []string{"a", "b"}},
		{"{{vars dup}} {{vars dup}}", []string{"dup"}},
		{"no vars here", []string{}},
		{"{{ vars spaced }}", []string{"spaced"}},
	}

	for _, tc := range tests {
		got := extractVarRefs(tc.input)
		if len(got) != len(tc.expected) {
			t.Errorf("extractVarRefs(%q): expected %v, got %v", tc.input, tc.expected, got)
			continue
		}
		for i, exp := range tc.expected {
			if got[i] != exp {
				t.Errorf("extractVarRefs(%q)[%d]: expected %q, got %q", tc.input, i, exp, got[i])
			}
		}
	}
}

func containsErrorMsg(errs []error, substr string) bool {
	for _, e := range errs {
		if strings.Contains(e.Error(), substr) {
			return true
		}
	}
	return false
}

func containsString(ss []string, substr string) bool {
	for _, s := range ss {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}
