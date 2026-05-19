package core

import (
	"fmt"
	"sort"
	"strings"
)

// TopologicalLayers returns unit names grouped into start-order layers.
// Units in the same layer have no dependency on each other and may start
// concurrently. Each layer must complete before the next one begins.
//
// Units listed in skipped are excluded; depends_on references to skipped
// units are treated as already satisfied so dependents can still run.
//
// Returns an error when a depends_on entry names an unknown unit or when
// the dependency graph contains a cycle.
func TopologicalLayers(units map[string]*RunpUnit, skipped map[string]bool) ([][]string, error) {
	inDegree, dependents, err := buildDependencyGraph(units, skipped)
	if err != nil {
		return nil, err
	}

	remaining := make(map[string]struct{}, len(inDegree))
	for name := range inDegree {
		remaining[name] = struct{}{}
	}

	var layers [][]string
	for len(remaining) > 0 {
		var layer []string
		for name := range remaining {
			if inDegree[name] == 0 {
				layer = append(layer, name)
			}
		}
		if len(layer) == 0 {
			return nil, cycleError(remaining)
		}
		sort.Strings(layer)
		layers = append(layers, layer)
		for _, name := range layer {
			delete(remaining, name)
			for _, dependent := range dependents[name] {
				inDegree[dependent]--
			}
		}
	}

	return layers, nil
}

// buildDependencyGraph computes in-degree counts and reverse-dependency edges
// for all active (non-skipped) units.
func buildDependencyGraph(units map[string]*RunpUnit, skipped map[string]bool) (map[string]int, map[string][]string, error) {
	inDegree := make(map[string]int, len(units))
	dependents := make(map[string][]string)

	for name := range units {
		if !skipped[name] {
			inDegree[name] = 0
		}
	}

	for name, unit := range units {
		if skipped[name] {
			continue
		}
		for _, dep := range unit.DependsOn {
			if _, exists := units[dep]; !exists {
				return nil, nil, fmt.Errorf("unit %q: depends_on references unknown unit %q", name, dep)
			}
			if skipped[dep] {
				continue
			}
			inDegree[name]++
			dependents[dep] = append(dependents[dep], name)
		}
	}

	return inDegree, dependents, nil
}

func cycleError(remaining map[string]struct{}) error {
	names := make([]string, 0, len(remaining))
	for name := range remaining {
		names = append(names, name)
	}
	sort.Strings(names)
	return fmt.Errorf("circular dependency detected among units: %s", strings.Join(names, ", "))
}

// validateDependsOn checks all depends_on references and detects cycles.
// It collects all unknown-reference errors before checking for cycles.
func validateDependsOn(rf *Runpfile) []error {
	var errs []error

	unitNames := make([]string, 0, len(rf.Units))
	for name := range rf.Units {
		unitNames = append(unitNames, name)
	}
	sort.Strings(unitNames)

	for _, name := range unitNames {
		unit := rf.Units[name]
		for _, dep := range unit.DependsOn {
			if _, exists := rf.Units[dep]; !exists {
				errs = append(errs, fmt.Errorf(
					"unit %q: depends_on references unknown unit %q", name, dep))
			}
		}
	}
	if len(errs) > 0 {
		return errs
	}

	_, err := TopologicalLayers(rf.Units, map[string]bool{})
	if err != nil {
		errs = append(errs, err)
	}
	return errs
}
