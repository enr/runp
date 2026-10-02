package core

import (
	"reflect"
	"testing"
)

func TestSplitCommandLine(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{`ls -l /library/`, []string{"ls", "-l", "/library/"}},
		{"echo \"Fowler collection created.\"\n", []string{"echo", "Fowler collection created."}},
		{`sh -c 'echo $HOME; id'`, []string{"sh", "-c", "echo $HOME; id"}},
		{`a\ b "c\"d" ''`, []string{"a b", `c"d`, ""}},
		{"  multi\n  line  ", []string{"multi", "line"}},
		{"", nil},
	}
	for _, tt := range tests {
		got, err := splitCommandLine(tt.in)
		if err != nil {
			t.Errorf("splitCommandLine(%q) error: %v", tt.in, err)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("splitCommandLine(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
	if _, err := splitCommandLine(`echo "unterminated`); err == nil {
		t.Error("expected error for unterminated quote")
	}
}

func TestContainerBuildArgs_NoShellInjection(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{Debug: false, Color: false})
	cp := &ContainerProcess{
		Image:   "alpine:{{vars tag}}",
		Ports:   []string{"{{vars port}}:80"},
		Env:     map[string]string{"B": "`id`; $(id) \"x", "A": "{{vars nested}}"},
		Command: `echo "hello world"`,
		vars: map[string]string{
			"tag":    "3.12",
			"port":   "8080",
			"nested": "{{vars tag}}",
		},
	}
	cp.SetID("web")
	got, err := cp.buildArgs()
	if err != nil {
		t.Fatalf("buildArgs: %v", err)
	}
	want := []string{
		"run", "-t", "--rm", "--name", "runp-web", "--network", "runp-network",
		"-p", "8080:80",
		// vars are expanded once: a value containing a var reference stays literal
		"-e", "A={{vars tag}}",
		"-e", "B=`id`; $(id) \"x",
		"alpine:3.12", "echo", "hello world",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildArgs()\n got  %q\n want %q", got, want)
	}
}
