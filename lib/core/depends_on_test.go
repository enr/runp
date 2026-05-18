package core

import (
	"strings"
	"testing"
)

func makeUnit(name string, deps ...string) *RunpUnit {
	return &RunpUnit{Name: name, Host: &HostProcess{}, DependsOn: deps}
}

func TestTopologicalLayers_NoDeps(t *testing.T) {
	units := map[string]*RunpUnit{
		"a": makeUnit("a"),
		"b": makeUnit("b"),
		"c": makeUnit("c"),
	}
	layers, err := TopologicalLayers(units, map[string]bool{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(layers) != 1 {
		t.Fatalf("expected 1 layer, got %d: %v", len(layers), layers)
	}
	if len(layers[0]) != 3 {
		t.Errorf("expected 3 units in layer 0, got %d", len(layers[0]))
	}
}

func TestTopologicalLayers_LinearChain(t *testing.T) {
	// a → b → c  (c depends on b, b depends on a)
	units := map[string]*RunpUnit{
		"a": makeUnit("a"),
		"b": makeUnit("b", "a"),
		"c": makeUnit("c", "b"),
	}
	layers, err := TopologicalLayers(units, map[string]bool{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(layers) != 3 {
		t.Fatalf("expected 3 layers, got %d: %v", len(layers), layers)
	}
	if layers[0][0] != "a" {
		t.Errorf("layer 0 should be [a], got %v", layers[0])
	}
	if layers[1][0] != "b" {
		t.Errorf("layer 1 should be [b], got %v", layers[1])
	}
	if layers[2][0] != "c" {
		t.Errorf("layer 2 should be [c], got %v", layers[2])
	}
}

func TestTopologicalLayers_DiamondDep(t *testing.T) {
	// a → b, a → c, b → d, c → d
	units := map[string]*RunpUnit{
		"a": makeUnit("a"),
		"b": makeUnit("b", "a"),
		"c": makeUnit("c", "a"),
		"d": makeUnit("d", "b", "c"),
	}
	layers, err := TopologicalLayers(units, map[string]bool{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Expected: layer0=[a], layer1=[b,c], layer2=[d]
	if len(layers) != 3 {
		t.Fatalf("expected 3 layers, got %d: %v", len(layers), layers)
	}
	if len(layers[0]) != 1 || layers[0][0] != "a" {
		t.Errorf("layer 0 should be [a], got %v", layers[0])
	}
	if len(layers[1]) != 2 {
		t.Errorf("layer 1 should have 2 units, got %v", layers[1])
	}
	if len(layers[2]) != 1 || layers[2][0] != "d" {
		t.Errorf("layer 2 should be [d], got %v", layers[2])
	}
}

func TestTopologicalLayers_Cycle(t *testing.T) {
	units := map[string]*RunpUnit{
		"a": makeUnit("a", "c"),
		"b": makeUnit("b", "a"),
		"c": makeUnit("c", "b"),
	}
	_, err := TopologicalLayers(units, map[string]bool{})
	if err == nil {
		t.Fatal("expected error for cycle, got nil")
	}
	if !strings.Contains(err.Error(), "circular dependency") {
		t.Errorf("error should mention 'circular dependency', got: %v", err)
	}
}

func TestTopologicalLayers_SelfDep(t *testing.T) {
	units := map[string]*RunpUnit{
		"a": makeUnit("a", "a"),
	}
	_, err := TopologicalLayers(units, map[string]bool{})
	if err == nil {
		t.Fatal("expected error for self-dependency, got nil")
	}
}

func TestTopologicalLayers_UnknownRef(t *testing.T) {
	units := map[string]*RunpUnit{
		"a": makeUnit("a", "nonexistent"),
	}
	_, err := TopologicalLayers(units, map[string]bool{})
	if err == nil {
		t.Fatal("expected error for unknown dep, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention the unknown unit, got: %v", err)
	}
}

func TestTopologicalLayers_SkippedDepTreatedAsSatisfied(t *testing.T) {
	// b depends on a, but a is skipped — b should still appear in layer 0.
	units := map[string]*RunpUnit{
		"a": makeUnit("a"),
		"b": makeUnit("b", "a"),
	}
	skipped := map[string]bool{"a": true}
	layers, err := TopologicalLayers(units, skipped)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(layers) != 1 {
		t.Fatalf("expected 1 layer (only b), got %d: %v", len(layers), layers)
	}
	if layers[0][0] != "b" {
		t.Errorf("expected b in layer 0, got %v", layers[0])
	}
}

func TestTopologicalLayers_EmptyUnits(t *testing.T) {
	layers, err := TopologicalLayers(map[string]*RunpUnit{}, map[string]bool{})
	if err != nil {
		t.Fatalf("unexpected error for empty units: %v", err)
	}
	if len(layers) != 0 {
		t.Errorf("expected 0 layers for empty units, got %d", len(layers))
	}
}

func TestValidateDependsOn_UnknownRef(t *testing.T) {
	rf := &Runpfile{
		Units: map[string]*RunpUnit{
			"a": makeUnit("a", "ghost"),
		},
	}
	errs := validateDependsOn(rf)
	if len(errs) == 0 {
		t.Fatal("expected validation error for unknown dep")
	}
	if !strings.Contains(errs[0].Error(), "ghost") {
		t.Errorf("error should mention 'ghost', got: %v", errs[0])
	}
}

func TestValidateDependsOn_Cycle(t *testing.T) {
	rf := &Runpfile{
		Units: map[string]*RunpUnit{
			"a": makeUnit("a", "b"),
			"b": makeUnit("b", "a"),
		},
	}
	errs := validateDependsOn(rf)
	if len(errs) == 0 {
		t.Fatal("expected validation error for cycle")
	}
	if !strings.Contains(errs[0].Error(), "circular") {
		t.Errorf("error should mention 'circular', got: %v", errs[0])
	}
}

func TestValidateDependsOn_Valid(t *testing.T) {
	rf := &Runpfile{
		Units: map[string]*RunpUnit{
			"a": makeUnit("a"),
			"b": makeUnit("b", "a"),
		},
	}
	errs := validateDependsOn(rf)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestValidateDependsOn_MultipleUnknownRefs(t *testing.T) {
	rf := &Runpfile{
		Units: map[string]*RunpUnit{
			"a": makeUnit("a", "x", "y"),
		},
	}
	errs := validateDependsOn(rf)
	if len(errs) != 2 {
		t.Errorf("expected 2 errors (one per unknown ref), got %d: %v", len(errs), errs)
	}
}
