package core

import (
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
	input := `{{ vars runp_workdir }}/config`
	expected := `/tmp/config`
	vars := map[string]string{
		"runp_workdir": "/tmp",
	}
	cliPreprocessor := newCliPreprocessor(vars)
	actual := cliPreprocessor.process(input)
	if actual != expected {
		t.Errorf("Expected output '%s', got '%s'\n", expected, actual)
	}
}
