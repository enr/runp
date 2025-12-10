package core

import (
	"fmt"
	"os"

	yaml "gopkg.in/yaml.v3"
)

var defaultEnvReader = func(name string) string {
	return os.Getenv(name)
}

// EnvVarCondition represents the type of check to perform on an environment variable
type EnvVarCondition string

const (
	// EnvVarConditionIsSet verifies that the variable exists and is not empty
	EnvVarConditionIsSet EnvVarCondition = "is_set"
	// EnvVarConditionIsUnset verifies that the variable does NOT exist or is empty
	EnvVarConditionIsUnset EnvVarCondition = "is_unset"
	// EnvVarConditionIsEqual verifies that the variable has a specific value
	EnvVarConditionIsEqual EnvVarCondition = "is_equal"
)

// EnvVarCheck represents a single check on an environment variable
type EnvVarCheck struct {
	Name      string          `yaml:"name"`
	Condition EnvVarCondition `yaml:"condition"`
	Value     string          `yaml:"value,omitempty"`
}

// EnvVarsPrecondition verifies environment variables
type EnvVarsPrecondition struct {
	// EnvVars is the list of environment variable checks
	EnvVars []EnvVarCheck

	envReader func(string) string
}

// UnmarshalYAML implements custom YAML unmarshaling to support inline list
func (p *EnvVarsPrecondition) UnmarshalYAML(value *yaml.Node) error {
	// When env_vars is a list directly under preconditions:
	// preconditions:
	//   env_vars:
	//     - name: ...
	// The value node will be a SequenceNode (list)
	if value.Kind == yaml.SequenceNode {
		var checks []EnvVarCheck
		if err := value.Decode(&checks); err != nil {
			return err
		}
		p.EnvVars = checks
		return nil
	}
	// If it's a mapping node, try to decode as a struct with env_vars field
	if value.Kind == yaml.MappingNode {
		type alias struct {
			EnvVars []EnvVarCheck `yaml:"env_vars"`
		}
		var a alias
		if err := value.Decode(&a); err != nil {
			return err
		}
		p.EnvVars = a.EnvVars
		return nil
	}
	// Default: try to decode directly (shouldn't happen in normal cases)
	return value.Decode(p)
}

// IsSet returns true if the precondition is configured
func (p *EnvVarsPrecondition) IsSet() bool {
	return len(p.EnvVars) > 0
}

// Verify checks all environment variable checks
func (p *EnvVarsPrecondition) Verify() PreconditionVerifyResult {
	if p.envReader == nil {
		p.envReader = defaultEnvReader
	}

	var reasons []string
	allPassed := true

	for _, check := range p.EnvVars {
		// Validate check configuration
		if check.Name == "" {
			reasons = append(reasons, "Environment variable check missing 'name' field")
			allPassed = false
			continue
		}

		if check.Condition == "" {
			reasons = append(reasons, fmt.Sprintf("Environment variable '%s' missing 'condition' field", check.Name))
			allPassed = false
			continue
		}

		// Validate that value is provided for is_equal condition
		if check.Condition == EnvVarConditionIsEqual && check.Value == "" {
			reasons = append(reasons, fmt.Sprintf("Environment variable '%s' requires 'value' field when using condition 'is_equal'", check.Name))
			allPassed = false
			continue
		}

		// Validate condition is one of the allowed values
		if check.Condition != EnvVarConditionIsSet &&
			check.Condition != EnvVarConditionIsUnset &&
			check.Condition != EnvVarConditionIsEqual {
			reasons = append(reasons, fmt.Sprintf("Invalid condition '%s' for environment variable '%s'", check.Condition, check.Name))
			allPassed = false
			continue
		}

		// Perform the check
		currentValue := p.envReader(check.Name)

		switch check.Condition {
		case EnvVarConditionIsSet:
			if currentValue == "" {
				reasons = append(reasons, fmt.Sprintf("Environment variable '%s' is not set", check.Name))
				allPassed = false
			}

		case EnvVarConditionIsUnset:
			if currentValue != "" {
				reasons = append(reasons, fmt.Sprintf("Environment variable '%s' is set but should be unset (current value: '%s')", check.Name, currentValue))
				allPassed = false
			}

		case EnvVarConditionIsEqual:
			if currentValue == "" {
				reasons = append(reasons, fmt.Sprintf("Environment variable '%s' is not set", check.Name))
				allPassed = false
			} else if currentValue != check.Value {
				reasons = append(reasons, fmt.Sprintf("Environment variable '%s' has value '%s' but expected '%s'", check.Name, currentValue, check.Value))
				allPassed = false
			}
		}
	}

	if allPassed {
		return PreconditionVerifyResult{
			Vote:    Proceed,
			Reasons: []string{},
		}
	}

	return PreconditionVerifyResult{
		Vote:    Stop,
		Reasons: reasons,
	}
}
