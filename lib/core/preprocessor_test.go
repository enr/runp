package core

import (
	"strings"
	"testing"
)

func TestProcessString(t *testing.T) {
	input := `{{vars foo}}{{ vars foo }}{{   vars foo}}{{vars foo	}}`
	expected := `barbarbarbar`
	vars := map[string]string{
		"foo": "bar",
	}
	cliPreprocessor := newCliPreprocessor(vars)
	actual := cliPreprocessor.process(input)
	if actual != expected {
		t.Errorf("Expected output '%s', got '%s'\n", expected, actual)
	}
}

func TestProcessString02(t *testing.T) {
	input := `{{ vars runp_workdir }}/config {{vars test_dir}}`
	expected := `/tmp/config .`
	vars := map[string]string{
		"runp_workdir": "/tmp",
		"test_dir":     ".",
	}
	cliPreprocessor := newCliPreprocessor(vars)
	actual := cliPreprocessor.process(input)
	if actual != expected {
		t.Errorf("Expected output '%s', got '%s'\n", expected, actual)
	}
}

func TestProcessUndeclaredVar(t *testing.T) {
	p := newCliPreprocessor(map[string]string{"foo": "bar"})
	got := p.process("{{vars missing}}")
	if !strings.Contains(got, "undefined:missing") {
		t.Errorf("expected undefined sentinel, got %q", got)
	}
}

func TestExpandVars_NoRefs(t *testing.T) {
	in := map[string]string{"a": "hello", "b": "world"}
	out, err := ExpandVars(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["a"] != "hello" || out["b"] != "world" {
		t.Errorf("unexpected result: %v", out)
	}
}

func TestExpandVars_VarInVar(t *testing.T) {
	in := map[string]string{
		"base": "/opt",
		"dir":  "{{vars base}}/app",
	}
	out, err := ExpandVars(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["dir"] != "/opt/app" {
		t.Errorf("expected /opt/app, got %q", out["dir"])
	}
}

func TestExpandVars_ChainedRefs(t *testing.T) {
	in := map[string]string{
		"a": "foo",
		"b": "{{vars a}}-bar",
		"c": "{{vars b}}-baz",
	}
	out, err := ExpandVars(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["c"] != "foo-bar-baz" {
		t.Errorf("expected foo-bar-baz, got %q", out["c"])
	}
}

func TestExpandVars_CircularRef(t *testing.T) {
	in := map[string]string{
		"a": "{{vars b}}",
		"b": "{{vars a}}",
	}
	_, err := ExpandVars(in)
	if err == nil {
		t.Fatal("expected error for circular reference, got nil")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("expected circular reference error, got: %v", err)
	}
}

func TestExpandVars_SelfRef(t *testing.T) {
	in := map[string]string{
		"a": "{{vars a}}",
	}
	_, err := ExpandVars(in)
	if err == nil {
		t.Fatal("expected error for self-reference, got nil")
	}
}

func TestExpandVars_UndeclaredRef(t *testing.T) {
	in := map[string]string{
		"a": "{{vars missing}}",
	}
	_, err := ExpandVars(in)
	if err == nil {
		t.Fatal("expected error for undeclared ref, got nil")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Errorf("expected error mentioning 'missing', got: %v", err)
	}
}

func TestExpandVars_EmptyMap(t *testing.T) {
	out, err := ExpandVars(map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected empty map, got %v", out)
	}
}
