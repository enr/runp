package core

import (
	"testing"

	yaml "gopkg.in/yaml.v3"
)

func TestEnvVarsPreconditionExists(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})

	mockEnv := map[string]string{
		"DATABASE_URL": "postgres://localhost:5432/mydb",
	}

	sut := &EnvVarsPrecondition{
		EnvVars: []EnvVarCheck{
			{
				Name:      "DATABASE_URL",
				Condition: EnvVarConditionIsSet,
			},
		},
		envReader: func(name string) string {
			return mockEnv[name]
		},
	}

	res := sut.Verify()
	if res.Vote != Proceed {
		t.Errorf("Expected Proceed but got %v: %v", res.Vote, res.Reasons)
	}
}

func TestEnvVarsPreconditionExistsEmpty(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})

	mockEnv := map[string]string{
		"DATABASE_URL": "",
	}

	sut := &EnvVarsPrecondition{
		EnvVars: []EnvVarCheck{
			{
				Name:      "DATABASE_URL",
				Condition: EnvVarConditionIsSet,
			},
		},
		envReader: func(name string) string {
			return mockEnv[name]
		},
	}

	res := sut.Verify()
	if res.Vote != Stop {
		t.Errorf("Expected Stop but got %v", res.Vote)
	}
	if len(res.Reasons) == 0 {
		t.Errorf("Expected error reasons but got none")
	}
}

func TestEnvVarsPreconditionNotExists(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})

	mockEnv := map[string]string{}

	sut := &EnvVarsPrecondition{
		EnvVars: []EnvVarCheck{
			{
				Name:      "DEBUG_MODE",
				Condition: EnvVarConditionIsUnset,
			},
		},
		envReader: func(name string) string {
			return mockEnv[name]
		},
	}

	res := sut.Verify()
	if res.Vote != Proceed {
		t.Errorf("Expected Proceed but got %v: %v", res.Vote, res.Reasons)
	}
}

func TestEnvVarsPreconditionNotExistsButExists(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})

	mockEnv := map[string]string{
		"DEBUG_MODE": "true",
	}

	sut := &EnvVarsPrecondition{
		EnvVars: []EnvVarCheck{
			{
				Name:      "DEBUG_MODE",
				Condition: EnvVarConditionIsUnset,
			},
		},
		envReader: func(name string) string {
			return mockEnv[name]
		},
	}

	res := sut.Verify()
	if res.Vote != Stop {
		t.Errorf("Expected Stop but got %v", res.Vote)
	}
	if len(res.Reasons) == 0 {
		t.Errorf("Expected error reasons but got none")
	}
}

func TestEnvVarsPreconditionEquals(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})

	mockEnv := map[string]string{
		"NODE_ENV": "production",
	}

	sut := &EnvVarsPrecondition{
		EnvVars: []EnvVarCheck{
			{
				Name:      "NODE_ENV",
				Condition: EnvVarConditionIsEqual,
				Value:     "production",
			},
		},
		envReader: func(name string) string {
			return mockEnv[name]
		},
	}

	res := sut.Verify()
	if res.Vote != Proceed {
		t.Errorf("Expected Proceed but got %v: %v", res.Vote, res.Reasons)
	}
}

func TestEnvVarsPreconditionEqualsMismatch(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})

	mockEnv := map[string]string{
		"NODE_ENV": "development",
	}

	sut := &EnvVarsPrecondition{
		EnvVars: []EnvVarCheck{
			{
				Name:      "NODE_ENV",
				Condition: EnvVarConditionIsEqual,
				Value:     "production",
			},
		},
		envReader: func(name string) string {
			return mockEnv[name]
		},
	}

	res := sut.Verify()
	if res.Vote != Stop {
		t.Errorf("Expected Stop but got %v", res.Vote)
	}
	if len(res.Reasons) == 0 {
		t.Errorf("Expected error reasons but got none")
	}
}

func TestEnvVarsPreconditionMultiple(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})

	mockEnv := map[string]string{
		"DATABASE_URL": "postgres://localhost:5432/mydb",
		"NODE_ENV":     "production",
	}

	sut := &EnvVarsPrecondition{
		EnvVars: []EnvVarCheck{
			{
				Name:      "DATABASE_URL",
				Condition: EnvVarConditionIsSet,
			},
			{
				Name:      "DEBUG_MODE",
				Condition: EnvVarConditionIsUnset,
			},
			{
				Name:      "NODE_ENV",
				Condition: EnvVarConditionIsEqual,
				Value:     "production",
			},
		},
		envReader: func(name string) string {
			return mockEnv[name]
		},
	}

	res := sut.Verify()
	if res.Vote != Proceed {
		t.Errorf("Expected Proceed but got %v: %v", res.Vote, res.Reasons)
	}
}

func TestEnvVarsPreconditionIsSet(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})

	sut := &EnvVarsPrecondition{
		EnvVars: []EnvVarCheck{
			{
				Name:      "DATABASE_URL",
				Condition: EnvVarConditionIsSet,
			},
		},
	}
	if !sut.IsSet() {
		t.Errorf("Expected IsSet() to return true")
	}

	sutEmpty := &EnvVarsPrecondition{
		EnvVars: []EnvVarCheck{},
	}
	if sutEmpty.IsSet() {
		t.Errorf("Expected IsSet() to return false for empty checks")
	}
}

func TestEnvVarsPreconditionInvalidCondition(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})

	mockEnv := map[string]string{
		"DATABASE_URL": "postgres://localhost:5432/mydb",
	}

	sut := &EnvVarsPrecondition{
		EnvVars: []EnvVarCheck{
			{
				Name:      "DATABASE_URL",
				Condition: EnvVarCondition("invalid_condition"),
			},
		},
		envReader: func(name string) string {
			return mockEnv[name]
		},
	}

	res := sut.Verify()
	if res.Vote != Stop {
		t.Errorf("Expected Stop but got %v", res.Vote)
	}
	if len(res.Reasons) == 0 {
		t.Errorf("Expected error reasons but got none")
	}
}

func TestEnvVarsPreconditionMissingName(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})

	sut := &EnvVarsPrecondition{
		EnvVars: []EnvVarCheck{
			{
				Name:      "",
				Condition: EnvVarConditionIsSet,
			},
		},
	}

	res := sut.Verify()
	if res.Vote != Stop {
		t.Errorf("Expected Stop but got %v", res.Vote)
	}
	if len(res.Reasons) == 0 {
		t.Errorf("Expected error reasons but got none")
	}
}

func TestEnvVarsPreconditionMissingValueForEquals(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})

	sut := &EnvVarsPrecondition{
		EnvVars: []EnvVarCheck{
			{
				Name:      "NODE_ENV",
				Condition: EnvVarConditionIsEqual,
				Value:     "",
			},
		},
	}

	res := sut.Verify()
	if res.Vote != Stop {
		t.Errorf("Expected Stop but got %v", res.Vote)
	}
	if len(res.Reasons) == 0 {
		t.Errorf("Expected error reasons but got none")
	}
}

func TestEnvVarsPreconditionEqualsNotSet(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})

	mockEnv := map[string]string{}

	sut := &EnvVarsPrecondition{
		EnvVars: []EnvVarCheck{
			{
				Name:      "API_KEY",
				Condition: EnvVarConditionIsEqual,
				Value:     "secret-key-123",
			},
		},
		envReader: func(name string) string {
			return mockEnv[name]
		},
	}

	res := sut.Verify()
	if res.Vote != Stop {
		t.Errorf("Expected Stop but got %v", res.Vote)
	}
	if len(res.Reasons) == 0 {
		t.Errorf("Expected error reasons but got none")
	}
}

func TestEnvVarsPreconditionMultipleWithFailures(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})

	mockEnv := map[string]string{
		"DATABASE_URL": "postgres://localhost:5432/mydb",
		"DEBUG_MODE":   "true",
		"NODE_ENV":     "development",
	}

	sut := &EnvVarsPrecondition{
		EnvVars: []EnvVarCheck{
			{
				Name:      "DATABASE_URL",
				Condition: EnvVarConditionIsSet,
			},
			{
				Name:      "DEBUG_MODE",
				Condition: EnvVarConditionIsUnset,
			},
			{
				Name:      "NODE_ENV",
				Condition: EnvVarConditionIsEqual,
				Value:     "production",
			},
			{
				Name:      "MISSING_VAR",
				Condition: EnvVarConditionIsSet,
			},
		},
		envReader: func(name string) string {
			return mockEnv[name]
		},
	}

	res := sut.Verify()
	if res.Vote != Stop {
		t.Errorf("Expected Stop but got %v", res.Vote)
	}
	if len(res.Reasons) < 3 {
		t.Errorf("Expected at least 3 error reasons but got %d: %v", len(res.Reasons), res.Reasons)
	}
}

func TestEnvVarsPreconditionYAMLParsing(t *testing.T) {
	yamlData := `
env_vars:
  - name: DATABASE_URL
    condition: is_set
  - name: DEBUG_MODE
    condition: is_unset
  - name: NODE_ENV
    condition: is_equal
    value: "production"
`
	var precondition EnvVarsPrecondition
	err := yaml.Unmarshal([]byte(yamlData), &precondition)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	if len(precondition.EnvVars) != 3 {
		t.Errorf("Expected 3 checks but got %d", len(precondition.EnvVars))
	}

	if precondition.EnvVars[0].Name != "DATABASE_URL" {
		t.Errorf("Expected first check name to be 'DATABASE_URL' but got '%s'", precondition.EnvVars[0].Name)
	}
	if precondition.EnvVars[0].Condition != EnvVarConditionIsSet {
		t.Errorf("Expected first check condition to be 'is_set' but got '%s'", precondition.EnvVars[0].Condition)
	}

	if precondition.EnvVars[1].Name != "DEBUG_MODE" {
		t.Errorf("Expected second check name to be 'DEBUG_MODE' but got '%s'", precondition.EnvVars[1].Name)
	}
	if precondition.EnvVars[1].Condition != EnvVarConditionIsUnset {
		t.Errorf("Expected second check condition to be 'is_unset' but got '%s'", precondition.EnvVars[1].Condition)
	}

	if precondition.EnvVars[2].Name != "NODE_ENV" {
		t.Errorf("Expected third check name to be 'NODE_ENV' but got '%s'", precondition.EnvVars[2].Name)
	}
	if precondition.EnvVars[2].Condition != EnvVarConditionIsEqual {
		t.Errorf("Expected third check condition to be 'is_equal' but got '%s'", precondition.EnvVars[2].Condition)
	}
	if precondition.EnvVars[2].Value != "production" {
		t.Errorf("Expected third check value to be 'production' but got '%s'", precondition.EnvVars[2].Value)
	}
}

func TestEnvVarsPreconditionYAMLParsingInPreconditions(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})
	runpfilePath := "../../testdata/runpfiles/preconditions-envvars.yml"
	rp, err := LoadRunpfileFromPath(runpfilePath)
	if err != nil {
		t.Fatalf("Runpfile %s, load error %v", runpfilePath, err)
	}
	isActuallyValid, _ := IsRunpfileValid(rp)
	if !isActuallyValid {
		t.Errorf("Expected runpfile valid but it is not\n")
	}
	units := rp.Units
	if len(units) != 1 {
		t.Fatalf("Expected units #%d but got #%d\n", 1, len(units))
	}

	unit := units["test1"]
	if unit == nil {
		t.Fatalf("Expected unit 'test1' not found")
	}

	if !unit.Preconditions.EnvVars.IsSet() {
		t.Errorf("Expected EnvVars precondition to be set, but got %d checks", len(unit.Preconditions.EnvVars.EnvVars))
	}

	if len(unit.Preconditions.EnvVars.EnvVars) != 2 {
		t.Errorf("Expected 2 checks but got %d", len(unit.Preconditions.EnvVars.EnvVars))
		return
	}

	// Verify the checks
	if unit.Preconditions.EnvVars.EnvVars[0].Name != "DATABASE_URL" {
		t.Errorf("Expected first check name to be 'DATABASE_URL' but got '%s'", unit.Preconditions.EnvVars.EnvVars[0].Name)
	}
	if unit.Preconditions.EnvVars.EnvVars[1].Name != "NODE_ENV" {
		t.Errorf("Expected second check name to be 'NODE_ENV' but got '%s'", unit.Preconditions.EnvVars.EnvVars[1].Name)
	}
	if unit.Preconditions.EnvVars.EnvVars[1].Value != "production" {
		t.Errorf("Expected second check value to be 'production' but got '%s'", unit.Preconditions.EnvVars.EnvVars[1].Value)
	}
}
