package qac

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// loadPlanFile reads, interpolates vars, strict-decodes and recursively
// processes include directives for the YAML plan at absPath.
// visiting holds the absolute paths currently on the include stack and is
// used to detect circular references.
func loadPlanFile(absPath string, visiting map[string]bool) (TestPlan, error) {
	if visiting[absPath] {
		return TestPlan{}, fmt.Errorf("circular include: %s", absPath)
	}
	visiting[absPath] = true
	defer delete(visiting, absPath)

	dat, err := os.ReadFile(absPath)
	if err != nil {
		return TestPlan{}, err
	}

	var planVars struct {
		Vars map[string]string `yaml:"vars"`
	}
	_ = yaml.Unmarshal(dat, &planVars)
	dat = interpolate(dat, planVars.Vars)

	var plan TestPlan
	dec := yaml.NewDecoder(bytes.NewReader(dat))
	dec.KnownFields(true)
	if err = dec.Decode(&plan); err != nil {
		return TestPlan{}, err
	}

	if len(plan.Include) == 0 {
		return plan, nil
	}

	basedir := filepath.Dir(absPath)
	for _, inc := range plan.Include {
		incPath := inc
		if !filepath.IsAbs(incPath) {
			incPath = filepath.Join(basedir, incPath)
		}
		incPlan, err := loadPlanFile(incPath, visiting)
		if err != nil {
			return TestPlan{}, fmt.Errorf("include %q: %w", inc, err)
		}
		mergePlan(&plan, incPlan)
	}

	return plan, nil
}

// mergePlan folds included into base. base always takes precedence on conflict.
//
// Merge semantics:
//   - vars: included vars become defaults; base vars override.
//   - preconditions: included checks are prepended (run first).
//   - setup: included commands are prepended (run first).
//   - teardown: included commands are appended (run last).
//   - specs: included specs that are not already in base are appended.
func mergePlan(base *TestPlan, included TestPlan) {
	for k, v := range included.Vars {
		if base.Vars == nil {
			base.Vars = make(map[string]string)
		}
		if _, exists := base.Vars[k]; !exists {
			base.Vars[k] = v
		}
	}

	base.Preconditions.FileSystemAssertions = append(
		included.Preconditions.FileSystemAssertions,
		base.Preconditions.FileSystemAssertions...,
	)

	base.Setup = append(included.Setup, base.Setup...)

	base.Teardown = append(base.Teardown, included.Teardown...)

	if base.Specs == nil && len(included.Specs) > 0 {
		base.Specs = make(map[string]Spec)
	}
	for _, key := range included.specOrder {
		if _, exists := base.Specs[key]; !exists {
			base.Specs[key] = included.Specs[key]
			base.specOrder = append(base.specOrder, key)
		}
	}
}
