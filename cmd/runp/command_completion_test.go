package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runCompletion runs the app in completion mode and returns the candidates.
func runCompletion(t *testing.T, args ...string) []string {
	t.Helper()
	argv := append([]string{"runp"}, args...)
	argv = append(argv, "--generate-bash-completion")
	// urfave/cli reads the word being completed from os.Args.
	oldArgs := os.Args
	os.Args = argv
	defer func() { os.Args = oldArgs }()
	out := captureStdout(t, func() {
		if err := newApp().Run(argv); err != nil {
			t.Errorf("Run(%v): %v", argv, err)
		}
	})
	return strings.Fields(out)
}

func TestCompletionCandidates(t *testing.T) {
	dir := t.TempDir()
	rf := filepath.Join(dir, "Runpfile")
	content := "units:\n  web:\n    host:\n      command: echo web\n  api:\n    host:\n      command: echo api\n"
	if err := os.WriteFile(rf, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		args []string
		want []string
		not  []string
	}{
		{"unit names", []string{"reload", "-f", rf}, []string{"api", "web"}, []string{"--file"}},
		{"unit names, global file flag", []string{"-f", rf, "reload"}, []string{"api", "web"}, nil},
		{"one unit only", []string{"reload", "-f", rf, "web"}, nil, []string{"api", "web"}},
		{"missing Runpfile", []string{"reload", "-f", filepath.Join(dir, "nope")}, nil, []string{"api"}},
		{"reload flags", []string{"reload", "-"}, []string{"--file", "--key-env"}, []string{"api"}},
		{"shells", []string{"completion"}, []string{"bash", "zsh", "fish"}, nil},
		{"setting keys get", []string{"config", "get"}, []string{"container_runner"}, nil},
		{"setting keys set", []string{"config", "set"}, []string{"container_runner"}, nil},
		{"setting value", []string{"config", "set", "container_runner"}, nil, []string{"container_runner"}},
		// "--" must not end flag parsing and run the command: the scripts pass "-".
		{"list flags", []string{"list", "-"}, []string{"--output", "--file"}, []string{"NAME"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runCompletion(t, tc.args...)
			for _, w := range tc.want {
				if !contains(got, w) {
					t.Errorf("candidates %v: missing %q", got, w)
				}
			}
			for _, w := range tc.not {
				if contains(got, w) {
					t.Errorf("candidates %v: unexpected %q", got, w)
				}
			}
			if len(tc.want) == 0 && len(got) > 0 && tc.not == nil {
				t.Errorf("expected no candidates, got %v", got)
			}
		})
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func TestDoCompletionScripts(t *testing.T) {
	for shell, marker := range map[string]string{
		"bash": "complete -o bashdefault -o default -F _runp_bash_completion runp",
		"zsh":  "compdef _runp_zsh_completion runp",
		"fish": "complete -c runp",
	} {
		out := captureStdout(t, func() {
			if err := newApp().Run([]string{"runp", "completion", shell}); err != nil {
				t.Errorf("completion %s: %v", shell, err)
			}
		})
		if !strings.Contains(out, marker) {
			t.Errorf("completion %s: output does not contain %q", shell, marker)
		}
	}
}

// The fish script is hand written: every command, alias and flag defined in
// the app must be mentioned in it.
func TestFishCompletionCoversCommands(t *testing.T) {
	app := newApp()
	for _, f := range app.Flags {
		checkFishFlag(t, "global", f.Names())
	}
	for _, cmd := range app.Commands {
		for _, name := range cmd.Names() {
			if !strings.Contains(fishCompletion, " "+name+" ") && !strings.Contains(fishCompletion, "-a "+name+" ") {
				t.Errorf("fish completion: command %q not found", name)
			}
		}
		for _, f := range cmd.Flags {
			checkFishFlag(t, cmd.Name, f.Names())
		}
		for _, sub := range cmd.Subcommands {
			if !strings.Contains(fishCompletion, "-a "+sub.Name+" ") {
				t.Errorf("fish completion: subcommand %s %s not found", cmd.Name, sub.Name)
			}
		}
	}
}

func checkFishFlag(t *testing.T, where string, names []string) {
	t.Helper()
	for _, n := range names {
		opt := "-l " + n + " "
		if len(n) == 1 {
			opt = "-s " + n + " "
		}
		if !strings.Contains(fishCompletion, opt) {
			t.Errorf("fish completion: %s flag %q not found", where, n)
		}
	}
}
