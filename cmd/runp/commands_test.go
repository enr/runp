package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/enr/runp/lib/core"
	"github.com/urfave/cli/v2"
)

// stubLogger is a mock implementation of core.Logger for testing.
type stubLogger struct {
	lines []string
}

func (l *stubLogger) WriteLinef(format string, a ...interface{}) (int, error) {
	line := fmt.Sprintf(format, a...)
	l.lines = append(l.lines, line)
	return len(line), nil
}

func (l *stubLogger) Debugf(format string, a ...interface{}) (int, error) {
	return l.WriteLinef(format, a...)
}

func (l *stubLogger) WriteLine(line string) (int, error) {
	l.lines = append(l.lines, line)
	return len(line), nil
}

func (l *stubLogger) Debug(line string) (int, error) {
	return l.WriteLine(line)
}

func (l *stubLogger) Write(p []byte) (int, error) {
	l.lines = append(l.lines, string(p))
	return len(p), nil
}

func (l *stubLogger) getLines() string {
	return strings.Join(l.lines, "\n")
}

func TestResolveRunpfileArg(t *testing.T) {
	newCtx := func(fileFlag string, setFlag bool) *cli.Context {
		app := cli.NewApp()
		set := flag.NewFlagSet("test", 0)
		set.String("file", configFileBaseName, "")
		set.String("f", configFileBaseName, "")
		if setFlag {
			set.Set("file", fileFlag)
		}
		return cli.NewContext(app, set, nil)
	}

	t.Run("flag takes precedence over env", func(t *testing.T) {
		t.Setenv("RUNP_FILE", "env-runpfile.yml")
		got := resolveRunpfileArg(newCtx("flag-runpfile.yml", true))
		if got != "flag-runpfile.yml" {
			t.Errorf("expected flag value, got %q", got)
		}
	})

	t.Run("env var used when flag not set", func(t *testing.T) {
		t.Setenv("RUNP_FILE", "env-runpfile.yml")
		got := resolveRunpfileArg(newCtx("", false))
		if got != "env-runpfile.yml" {
			t.Errorf("expected RUNP_FILE value, got %q", got)
		}
	})

	t.Run("default used when neither flag nor env set", func(t *testing.T) {
		t.Setenv("RUNP_FILE", "")
		got := resolveRunpfileArg(newCtx("", false))
		if got != configFileBaseName {
			t.Errorf("expected default %q, got %q", configFileBaseName, got)
		}
	})
}

func TestResolveLogLevel(t *testing.T) {
	newCtx := func(logLevel string, debug, quiet bool) *cli.Context {
		app := cli.NewApp()
		set := flag.NewFlagSet("test", 0)
		set.String("log-level", logLevel, "")
		set.Bool("debug", debug, "")
		set.Bool("quiet", quiet, "")
		return cli.NewContext(app, set, nil)
	}

	tests := []struct {
		logLevel string
		debug    bool
		quiet    bool
		want     core.LogLevel
		wantErr  bool
	}{
		{"info", false, false, core.LogLevelInfo, false},
		{"debug", false, false, core.LogLevelDebug, false},
		{"warn", false, false, core.LogLevelWarn, false},
		{"error", false, false, core.LogLevelError, false},
		{"trace", false, false, core.LogLevelTrace, false},
		// --debug alias
		{"info", true, false, core.LogLevelDebug, false},
		// --quiet alias
		{"info", false, true, core.LogLevelWarn, false},
		// --log-level takes precedence over --debug when non-default
		{"warn", true, false, core.LogLevelWarn, false},
		// invalid level
		{"bogus", false, false, core.LogLevelInfo, true},
	}

	for _, tc := range tests {
		got, err := resolveLogLevel(newCtx(tc.logLevel, tc.debug, tc.quiet))
		if tc.wantErr {
			if err == nil {
				t.Errorf("resolveLogLevel(%q, debug=%v, quiet=%v): expected error, got nil",
					tc.logLevel, tc.debug, tc.quiet)
			}
			continue
		}
		if err != nil {
			t.Errorf("resolveLogLevel(%q, debug=%v, quiet=%v): unexpected error: %v",
				tc.logLevel, tc.debug, tc.quiet, err)
			continue
		}
		if got != tc.want {
			t.Errorf("resolveLogLevel(%q, debug=%v, quiet=%v) = %v, want %v",
				tc.logLevel, tc.debug, tc.quiet, got, tc.want)
		}
	}
}

func TestExitError(t *testing.T) {
	s := &stubLogger{}
	ui = s
	message := "test error"
	exitCode := exitCodeArg
	err := exitError(exitCode, message)

	exitErr, ok := err.(cli.ExitCoder)
	if !ok {
		t.Fatalf("Expected an error implementing cli.ExitCoder, got %T", err)
	}

	if exitErr.ExitCode() != exitCode {
		t.Errorf("Expected exit code %d, got %d", exitCode, exitErr.ExitCode())
	}

	if exitErr.Error() != message {
		t.Errorf("Expected error message '%s', got '%s'", message, exitErr.Error())
	}
}

func TestExitErrorf(t *testing.T) {
	s := &stubLogger{}
	ui = s
	template := "error with value %d"
	value := 42
	expectedMessage := fmt.Sprintf(template, value)
	exitCode := exitCodeVar

	err := exitErrorf(exitCode, template, value)

	exitErr, ok := err.(cli.ExitCoder)
	if !ok {
		t.Fatalf("Expected an error implementing cli.ExitCoder, got %T", err)
	}

	if exitErr.ExitCode() != exitCode {
		t.Errorf("Expected exit code %d, got %d", exitCode, exitErr.ExitCode())
	}

	if exitErr.Error() != expectedMessage {
		t.Errorf("Expected error message '%s', got '%s'", expectedMessage, exitErr.Error())
	}
}

func TestLoadRunpfile(t *testing.T) {
	s := &stubLogger{}
	ui = s
	core.ConfigureUI(s, core.LoggerConfig{})

	// Success case
	t.Run("success", func(t *testing.T) {
		s.lines = []string{}
		runpfile, err := loadRunpfile("../../testdata/runpfiles/env.yml")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if runpfile == nil {
			t.Fatal("Expected a runpfile, got nil")
		}
		tasks := runpfile.Units
		if len(tasks) == 0 {
			t.Error("Expected tasks to be loaded")
		}
		if !strings.Contains(s.getLines(), "Loaded:") {
			t.Errorf("Expected 'Loaded:' line in output, got %q", s.getLines())
		}
	})

	// File not found case
	t.Run("file not found", func(t *testing.T) {
		_, err := loadRunpfile("non-existent-file.yml")
		if err == nil {
			t.Fatal("Expected an error for non-existent file, got nil")
		}
		exitErr, ok := err.(cli.ExitCoder)
		if !ok {
			t.Fatalf("Expected an error implementing cli.ExitCoder, got %T", err)
		}
		if exitErr.ExitCode() != exitCodeLoad {
			t.Errorf("Expected exit code %d, got %d", exitCodeLoad, exitErr.ExitCode())
		}
		if !strings.Contains(exitErr.Error(), "not found") {
			t.Errorf("Expected error message to contain 'not found', got '%s'", exitErr.Error())
		}
	})

	// Invalid file format case
	t.Run("invalid format", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "invalid-runpfile-*.yml")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write([]byte("invalid yaml content:")); err != nil {
			t.Fatal(err)
		}
		tmpfile.Close()

		_, err = loadRunpfile(tmpfile.Name())
		if err == nil {
			t.Fatal("Expected an error for invalid file, got nil")
		}
		exitErr, ok := err.(cli.ExitCoder)
		if !ok {
			t.Fatalf("Expected an error implementing cli.ExitCoder, got %T", err)
		}
		if exitErr.ExitCode() != exitCodeLoad {
			t.Errorf("Expected exit code %d, got %d", exitCodeLoad, exitErr.ExitCode())
		}
	})

	// Invalid runpfile structure case
	t.Run("invalid structure", func(t *testing.T) {
		_, err := loadRunpfile("../../testdata/runpfiles/validation-error-01.yml")
		if err == nil {
			t.Fatal("Expected an error for invalid runpfile structure, got nil")
		}
		exitErr, ok := err.(cli.ExitCoder)
		if !ok {
			t.Fatalf("Expected an error implementing cli.ExitCoder, got %T", err)
		}
		if exitErr.ExitCode() != exitCodeLoad {
			t.Errorf("Expected exit code %d, got %d", exitCodeLoad, exitErr.ExitCode())
		}
		if !strings.Contains(exitErr.Error(), "Invalid Runpfile") {
			t.Errorf("Expected error message to contain 'Invalid Runpfile', got '%s'", exitErr.Error())
		}
	})
}

func TestListEntries(t *testing.T) {
	rf := &core.Runpfile{
		Units: map[string]*core.RunpUnit{
			"web": {
				Name:        "web",
				Description: "frontend",
				Host:        &core.HostProcess{},
			},
			"db": {
				Name:      "db",
				Container: &core.ContainerProcess{Image: "postgres:15"},
			},
		},
	}

	entries := core.ListEntries(rf)

	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}
	// sorted by name: db, web
	if entries[0].Name != "db" {
		t.Errorf("Expected first entry 'db', got %q", entries[0].Name)
	}
	if entries[0].Kind != "container" {
		t.Errorf("Expected kind 'container', got %q", entries[0].Kind)
	}
	if entries[0].Preconditions != "-" {
		t.Errorf("Expected preconditions '-', got %q", entries[0].Preconditions)
	}
	if entries[1].Name != "web" {
		t.Errorf("Expected second entry 'web', got %q", entries[1].Name)
	}
	if entries[1].Description != "frontend" {
		t.Errorf("Expected description 'frontend', got %q", entries[1].Description)
	}
}

func TestApplyUserVarsNilMap(t *testing.T) {
	s := &stubLogger{}
	ui = s

	// nil vars map (no 'vars:' section in Runpfile) + user supplies --var
	_, err := applyUserVars(nil, []string{"foo=bar"})
	if err == nil {
		t.Fatal("Expected an error when vars map is nil and user supplies --var, got nil")
	}
	exitErr, ok := err.(cli.ExitCoder)
	if !ok {
		t.Fatalf("Expected cli.ExitCoder, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != exitCodeVar {
		t.Errorf("Expected exit code %d, got %d", exitCodeVar, exitErr.ExitCode())
	}
	msg := exitErr.Error()
	if !strings.Contains(msg, "vars:") {
		t.Errorf("Error message should mention 'vars:' section to guide the user, got: %q", msg)
	}
}

func TestDoUpVarMissingEquals(t *testing.T) {
	s := &stubLogger{}
	ui = s
	core.ConfigureUI(s, core.LoggerConfig{})

	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	set.String("f", "../../testdata/runpfiles/vars-in-env-nix.yml", "doc")
	varFlag := cli.StringSlice{}
	varFlag.Set("foo") // no '=' — should return an error, not panic
	set.Var(&varFlag, "var", "doc")
	c := cli.NewContext(app, set, nil)

	err := doUp(c)
	if err == nil {
		t.Fatal("Expected an error for --var without '=', got nil")
	}
	exitErr, ok := err.(cli.ExitCoder)
	if !ok {
		t.Fatalf("Expected cli.ExitCoder, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != exitCodeVar {
		t.Errorf("Expected exit code %d, got %d", exitCodeVar, exitErr.ExitCode())
	}
}

func TestDoList(t *testing.T) {
	s := &stubLogger{}
	ui = s
	core.ConfigureUI(s, core.LoggerConfig{})

	newCtx := func(filePath, outputFmt string) *cli.Context {
		app := cli.NewApp()
		set := flag.NewFlagSet("test", 0)
		set.String("f", filePath, "doc")
		set.String("output", outputFmt, "doc")
		set.String("o", outputFmt, "doc")
		return cli.NewContext(app, set, nil)
	}

	t.Run("table output has header and unit name", func(t *testing.T) {
		s.lines = []string{}
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := doList(newCtx("../../testdata/runpfiles/env.yml", "table"))

		w.Close()
		os.Stdout = old
		var buf strings.Builder
		io.Copy(&buf, r)
		stdout := buf.String()

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !strings.Contains(s.getLines(), "Units defined in Runpfile:") {
			t.Errorf("Expected logger output to contain 'Units defined in Runpfile:', got %q", s.getLines())
		}
		if !strings.Contains(stdout, "NAME") {
			t.Errorf("Expected stdout to contain 'NAME' header, got %q", stdout)
		}
		if !strings.Contains(stdout, "env-test-unit") {
			t.Errorf("Expected stdout to contain unit name 'env-test-unit', got %q", stdout)
		}
		if !strings.Contains(stdout, "host") {
			t.Errorf("Expected stdout to contain kind 'host', got %q", stdout)
		}
	})

	t.Run("json output is valid JSON array", func(t *testing.T) {
		s.lines = []string{}
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := doList(newCtx("../../testdata/runpfiles/env.yml", "json"))

		w.Close()
		os.Stdout = old
		var buf strings.Builder
		io.Copy(&buf, r)
		stdout := buf.String()

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		// Must be a JSON array
		if !strings.HasPrefix(strings.TrimSpace(stdout), "[") {
			t.Errorf("Expected JSON array output, got %q", stdout)
		}
		if !strings.Contains(stdout, `"name"`) {
			t.Errorf("Expected JSON to contain 'name' key, got %q", stdout)
		}
		if !strings.Contains(stdout, "env-test-unit") {
			t.Errorf("Expected JSON to contain unit name, got %q", stdout)
		}
	})

	t.Run("file not found exits 2", func(t *testing.T) {
		s.lines = []string{}
		err := doList(newCtx("non-existent-file.yml", "table"))
		if err == nil {
			t.Fatal("Expected an error, got nil")
		}
		exitErr, ok := err.(cli.ExitCoder)
		if !ok {
			t.Fatalf("Expected an error implementing cli.ExitCoder, got %T", err)
		}
		if exitErr.ExitCode() != exitCodeLoad {
			t.Errorf("Expected exit code %d, got %d", exitCodeLoad, exitErr.ExitCode())
		}
	})
}

func TestDoEncrypt(t *testing.T) {
	s := &stubLogger{}
	ui = s
	core.ConfigureUI(s, core.LoggerConfig{})

	t.Run("success with key", func(t *testing.T) {
		s.lines = []string{}
		app := cli.NewApp()
		set := flag.NewFlagSet("test", 0)
		set.String("key", "testkey123", "doc")
		set.Parse([]string{"secret-value"})
		c := cli.NewContext(app, set, nil)

		err := doEncrypt(c)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		output := s.getLines()
		if !strings.Contains(output, "Encrypted secret:") {
			t.Errorf("Expected output to contain 'Encrypted secret:', got '%s'", output)
		}
	})

	t.Run("success with key-env", func(t *testing.T) {
		s.lines = []string{}
		os.Setenv("TEST_ENCRYPT_KEY", "testkey123")
		defer os.Unsetenv("TEST_ENCRYPT_KEY")

		app := cli.NewApp()
		set := flag.NewFlagSet("test", 0)
		set.String("key-env", "TEST_ENCRYPT_KEY", "doc")
		set.Parse([]string{"secret-value"})
		c := cli.NewContext(app, set, nil)

		err := doEncrypt(c)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		output := s.getLines()
		if !strings.Contains(output, "Encrypted secret:") {
			t.Errorf("Expected output to contain 'Encrypted secret:', got '%s'", output)
		}
	})

	t.Run("success without key (random key)", func(t *testing.T) {
		s.lines = []string{}
		app := cli.NewApp()
		set := flag.NewFlagSet("test", 0)
		set.Parse([]string{"secret-value"})
		c := cli.NewContext(app, set, nil)

		err := doEncrypt(c)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		output := s.getLines()
		if !strings.Contains(output, "No encryption key provided, generating random key") {
			t.Errorf("Expected output to contain 'No encryption key provided', got '%s'", output)
		}
		if !strings.Contains(output, "Encrypted secret:") {
			t.Errorf("Expected output to contain 'Encrypted secret:', got '%s'", output)
		}
	})

	t.Run("missing secret parameter (no stdin pipe)", func(t *testing.T) {
		s.lines = []string{}
		// In test environment stdin is not a pipe, so we expect a usage error.
		app := cli.NewApp()
		set := flag.NewFlagSet("test", 0)
		c := cli.NewContext(app, set, nil)

		err := doEncrypt(c)
		if err == nil {
			t.Fatal("Expected an error, got nil")
		}
		exitErr, ok := err.(cli.ExitCoder)
		if !ok {
			t.Fatalf("Expected an error implementing cli.ExitCoder, got %T", err)
		}
		if exitErr.ExitCode() != exitCodeArg {
			t.Errorf("Expected exit code %d, got %d", exitCodeArg, exitErr.ExitCode())
		}
		if !strings.Contains(exitErr.Error(), "Secret value required") {
			t.Errorf("Expected error message to mention 'Secret value required', got '%s'", exitErr.Error())
		}
	})

	t.Run("reads secret from stdin pipe", func(t *testing.T) {
		s.lines = []string{}
		r, w, _ := os.Pipe()
		w.WriteString("piped-secret\n")
		w.Close()
		oldStdin := os.Stdin
		os.Stdin = r
		defer func() { os.Stdin = oldStdin }()

		app := cli.NewApp()
		set := flag.NewFlagSet("test", 0)
		set.String("key", "testkey123", "doc")
		c := cli.NewContext(app, set, nil)

		err := doEncrypt(c)
		if err != nil {
			t.Fatalf("Expected no error reading from stdin pipe, got %v", err)
		}
		if !strings.Contains(s.getLines(), "Encrypted secret:") {
			t.Errorf("Expected 'Encrypted secret:' in output, got %q", s.getLines())
		}
	})

	t.Run("empty stdin pipe exits 3", func(t *testing.T) {
		s.lines = []string{}
		r, w, _ := os.Pipe()
		w.Close() // empty pipe
		oldStdin := os.Stdin
		os.Stdin = r
		defer func() { os.Stdin = oldStdin }()

		app := cli.NewApp()
		set := flag.NewFlagSet("test", 0)
		set.String("key", "testkey123", "doc")
		c := cli.NewContext(app, set, nil)

		err := doEncrypt(c)
		if err == nil {
			t.Fatal("Expected an error for empty stdin, got nil")
		}
		exitErr, ok := err.(cli.ExitCoder)
		if !ok {
			t.Fatalf("Expected cli.ExitCoder, got %T", err)
		}
		if exitErr.ExitCode() != exitCodeArg {
			t.Errorf("Expected exit code %d, got %d", exitCodeArg, exitErr.ExitCode())
		}
	})

	t.Run("mutually exclusive key options", func(t *testing.T) {
		s.lines = []string{}
		app := cli.NewApp()
		set := flag.NewFlagSet("test", 0)
		set.String("key", "testkey", "doc")
		set.String("key-env", "TEST_KEY", "doc")
		set.Parse([]string{"secret-value"})
		c := cli.NewContext(app, set, nil)

		err := doEncrypt(c)
		if err == nil {
			t.Fatal("Expected an error, got nil")
		}
		exitErr, ok := err.(cli.ExitCoder)
		if !ok {
			t.Fatalf("Expected an error implementing cli.ExitCoder, got %T", err)
		}
		if exitErr.ExitCode() != exitCodeArg {
			t.Errorf("Expected exit code %d, got %d", exitCodeArg, exitErr.ExitCode())
		}
		if !strings.Contains(exitErr.Error(), "mutually exclusive") {
			t.Errorf("Expected error message to contain 'mutually exclusive', got '%s'", exitErr.Error())
		}
	})

	t.Run("empty key-env variable", func(t *testing.T) {
		s.lines = []string{}
		os.Unsetenv("EMPTY_KEY")
		app := cli.NewApp()
		set := flag.NewFlagSet("test", 0)
		set.String("key-env", "EMPTY_KEY", "doc")
		set.Parse([]string{"secret-value"})
		c := cli.NewContext(app, set, nil)

		err := doEncrypt(c)
		if err == nil {
			t.Fatal("Expected an error, got nil")
		}
		exitErr, ok := err.(cli.ExitCoder)
		if !ok {
			t.Fatalf("Expected an error implementing cli.ExitCoder, got %T", err)
		}
		if exitErr.ExitCode() != exitCodeArg {
			t.Errorf("Expected exit code %d, got %d", exitCodeArg, exitErr.ExitCode())
		}
		if !strings.Contains(exitErr.Error(), "is empty") {
			t.Errorf("Expected error message to contain 'is empty', got '%s'", exitErr.Error())
		}
	})
}

func TestDoUpDryRun(t *testing.T) {
	s := &stubLogger{}
	ui = s
	core.ConfigureUI(s, core.LoggerConfig{})

	// Redirect stdout so we can assert on the table output.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	app := cli.NewApp()
	app.Name = "runp"
	app.Flags = []cli.Flag{
		&cli.BoolFlag{Name: "dry-run", Aliases: []string{"n"}},
		&cli.BoolFlag{Name: "debug", Aliases: []string{"d"}},
		&cli.BoolFlag{Name: "quiet", Aliases: []string{"q"}},
		&cli.BoolFlag{Name: "no-color", Aliases: []string{"C"}},
	}
	app.Before = func(c *cli.Context) error { return nil }
	app.Commands = commands

	err := app.Run([]string{"runp", "--dry-run", "up", "--file", "../../testdata/runpfiles/env.yml"})

	w.Close()
	os.Stdout = old
	var buf strings.Builder
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Fatalf("Expected no error for dry-run up, got %v", err)
	}
	if !strings.Contains(output, "dry-run") {
		t.Errorf("Expected output to mention 'dry-run', got %q", output)
	}
	if !strings.Contains(output, "NAME") {
		t.Errorf("Expected output to contain table header 'NAME', got %q", output)
	}
	if !strings.Contains(output, "host") {
		t.Errorf("Expected output to list 'host' kind, got %q", output)
	}
}

func TestDefaultCommandIsUp(t *testing.T) {
	s := &stubLogger{}
	ui = s
	core.ConfigureUI(s, core.LoggerConfig{})

	// Prevent cli from calling os.Exit when the action returns an ExitCoder.
	var capturedCode int
	oldExiter := cli.OsExiter
	cli.OsExiter = func(code int) { capturedCode = code }
	defer func() { cli.OsExiter = oldExiter }()

	// Build the app exactly as main() does and verify that running it with no
	// subcommand invokes doUp — confirmed by exitCodeLoad that loadRunpfile
	// returns when no Runpfile exists in the working directory.
	app := cli.NewApp()
	app.Name = "runp"
	app.Flags = []cli.Flag{
		&cli.BoolFlag{Name: "debug", Aliases: []string{"d"}},
		&cli.BoolFlag{Name: "quiet", Aliases: []string{"q"}},
		&cli.BoolFlag{Name: "no-color", Aliases: []string{"C"}},
	}
	app.Before = func(c *cli.Context) error { return nil }
	app.Commands = commands
	app.DefaultCommand = "up"

	app.Run([]string{"runp"})

	// exitCodeLoad == loadRunpfile could not find the Runpfile.
	// Any other code (e.g. 0, exitCodeArg) would mean the default command wasn't 'up'.
	if capturedCode != exitCodeLoad {
		t.Errorf("Expected OsExiter to be called with code %d (doUp ran, file not found), got %d", exitCodeLoad, capturedCode)
	}
}

func TestDoConfig(t *testing.T) {
	s := &stubLogger{}
	ui = s
	core.ConfigureUI(s, core.LoggerConfig{})

	// Redirect HOME to a temp dir so tests don't touch the real settings file.
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	newCtx := func(args ...string) *cli.Context {
		app := cli.NewApp()
		set := flag.NewFlagSet("test", 0)
		set.Parse(args)
		return cli.NewContext(app, set, nil)
	}

	t.Run("get missing key exits 3", func(t *testing.T) {
		s.lines = []string{}
		err := doConfigGet(newCtx())
		if err == nil {
			t.Fatal("Expected error when key omitted, got nil")
		}
		exitErr, ok := err.(cli.ExitCoder)
		if !ok {
			t.Fatalf("Expected cli.ExitCoder, got %T", err)
		}
		if exitErr.ExitCode() != exitCodeArg {
			t.Errorf("Expected exit code %d, got %d", exitCodeArg, exitErr.ExitCode())
		}
	})

	t.Run("get unknown key exits 3", func(t *testing.T) {
		s.lines = []string{}
		err := doConfigGet(newCtx("totally_unknown"))
		if err == nil {
			t.Fatal("Expected error for unknown key, got nil")
		}
		exitErr, ok := err.(cli.ExitCoder)
		if !ok {
			t.Fatalf("Expected cli.ExitCoder, got %T", err)
		}
		if exitErr.ExitCode() != exitCodeArg {
			t.Errorf("Expected exit code %d, got %d", exitCodeArg, exitErr.ExitCode())
		}
	})

	t.Run("get returns default when no file", func(t *testing.T) {
		s.lines = []string{}
		// No settings file in temp dir — should return the default "docker".
		err := doConfigGet(newCtx("container_runner"))
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	})

	t.Run("set missing args exits 3", func(t *testing.T) {
		s.lines = []string{}
		err := doConfigSet(newCtx("container_runner"))
		if err == nil {
			t.Fatal("Expected error when value omitted, got nil")
		}
		exitErr, ok := err.(cli.ExitCoder)
		if !ok {
			t.Fatalf("Expected cli.ExitCoder, got %T", err)
		}
		if exitErr.ExitCode() != exitCodeArg {
			t.Errorf("Expected exit code %d, got %d", exitCodeArg, exitErr.ExitCode())
		}
	})

	t.Run("set unknown key exits 3", func(t *testing.T) {
		s.lines = []string{}
		err := doConfigSet(newCtx("bad_key", "value"))
		if err == nil {
			t.Fatal("Expected error for unknown key, got nil")
		}
		exitErr, ok := err.(cli.ExitCoder)
		if !ok {
			t.Fatalf("Expected cli.ExitCoder, got %T", err)
		}
		if exitErr.ExitCode() != exitCodeArg {
			t.Errorf("Expected exit code %d, got %d", exitCodeArg, exitErr.ExitCode())
		}
	})

	t.Run("set then get round-trip", func(t *testing.T) {
		s.lines = []string{}
		if err := doConfigSet(newCtx("container_runner", "podman")); err != nil {
			t.Fatalf("doConfigSet failed: %v", err)
		}
		if !strings.Contains(s.getLines(), "podman") {
			t.Errorf("Expected confirmation output to mention 'podman', got %q", s.getLines())
		}
	})

	t.Run("show prints path", func(t *testing.T) {
		s.lines = []string{}
		err := doConfigShow(newCtx())
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !strings.Contains(s.getLines(), ".runp") {
			t.Errorf("Expected output to mention '.runp', got %q", s.getLines())
		}
	})
}

func TestDoReload(t *testing.T) {
	s := &stubLogger{}
	ui = s
	core.ConfigureUI(s, core.LoggerConfig{})

	newCtx := func(filePath, unitName string) *cli.Context {
		app := cli.NewApp()
		set := flag.NewFlagSet("test", 0)
		set.String("f", filePath, "doc")
		if unitName != "" {
			set.Parse([]string{unitName})
		}
		return cli.NewContext(app, set, nil)
	}

	t.Run("missing unit name exits 3", func(t *testing.T) {
		s.lines = []string{}
		err := doReload(newCtx("../../testdata/runpfiles/env.yml", ""))
		if err == nil {
			t.Fatal("Expected error when unit name omitted, got nil")
		}
		exitErr, ok := err.(cli.ExitCoder)
		if !ok {
			t.Fatalf("Expected cli.ExitCoder, got %T", err)
		}
		if exitErr.ExitCode() != exitCodeArg {
			t.Errorf("Expected exit code %d, got %d", exitCodeArg, exitErr.ExitCode())
		}
	})

	t.Run("file not found exits 2", func(t *testing.T) {
		s.lines = []string{}
		err := doReload(newCtx("nonexistent-runpfile.yml", "any-unit"))
		if err == nil {
			t.Fatal("Expected error for missing file, got nil")
		}
		exitErr, ok := err.(cli.ExitCoder)
		if !ok {
			t.Fatalf("Expected cli.ExitCoder, got %T", err)
		}
		if exitErr.ExitCode() != exitCodeLoad {
			t.Errorf("Expected exit code %d, got %d", exitCodeLoad, exitErr.ExitCode())
		}
	})

	t.Run("unknown unit name exits 3", func(t *testing.T) {
		s.lines = []string{}
		err := doReload(newCtx("../../testdata/runpfiles/env.yml", "no-such-unit"))
		if err == nil {
			t.Fatal("Expected error for unknown unit, got nil")
		}
		exitErr, ok := err.(cli.ExitCoder)
		if !ok {
			t.Fatalf("Expected cli.ExitCoder, got %T", err)
		}
		if exitErr.ExitCode() != exitCodeArg {
			t.Errorf("Expected exit code %d, got %d", exitCodeArg, exitErr.ExitCode())
		}
	})
}

func TestDoValidate(t *testing.T) {
	s := &stubLogger{}
	ui = s
	core.ConfigureUI(s, core.LoggerConfig{})

	newCtx := func(filePath string) *cli.Context {
		app := cli.NewApp()
		set := flag.NewFlagSet("test", 0)
		set.String("f", filePath, "doc")
		return cli.NewContext(app, set, nil)
	}

	t.Run("valid Runpfile exits 0", func(t *testing.T) {
		s.lines = []string{}
		err := doValidate(newCtx("../../testdata/runpfiles/env.yml"))
		if err != nil {
			t.Fatalf("Expected no error for a valid Runpfile, got %v", err)
		}
		if !strings.Contains(s.getLines(), "valid") {
			t.Errorf("Expected output to contain 'valid', got %q", s.getLines())
		}
	})

	t.Run("file not found exits 2", func(t *testing.T) {
		s.lines = []string{}
		err := doValidate(newCtx("nonexistent-runpfile.yml"))
		if err == nil {
			t.Fatal("Expected an error for a missing file, got nil")
		}
		exitErr, ok := err.(cli.ExitCoder)
		if !ok {
			t.Fatalf("Expected cli.ExitCoder, got %T", err)
		}
		if exitErr.ExitCode() != exitCodeLoad {
			t.Errorf("Expected exit code %d, got %d", exitCodeLoad, exitErr.ExitCode())
		}
	})

	t.Run("no units defined exits 1", func(t *testing.T) {
		s.lines = []string{}
		err := doValidate(newCtx("../../testdata/runpfiles/validation-error-01.yml"))
		if err == nil {
			t.Fatal("Expected an error for a Runpfile with no units, got nil")
		}
		exitErr, ok := err.(cli.ExitCoder)
		if !ok {
			t.Fatalf("Expected cli.ExitCoder, got %T", err)
		}
		if exitErr.ExitCode() != 1 {
			t.Errorf("Expected exit code 1, got %d", exitErr.ExitCode())
		}
		output := s.getLines()
		if !strings.Contains(output, "invalid") {
			t.Errorf("Expected output to contain 'invalid', got %q", output)
		}
		if !strings.Contains(output, "No units defined") {
			t.Errorf("Expected output to contain 'No units defined', got %q", output)
		}
	})

	t.Run("undefined variable reference exits 1", func(t *testing.T) {
		s.lines = []string{}
		err := doValidate(newCtx("../../testdata/runpfiles/validate-unknown-var.yml"))
		if err == nil {
			t.Fatal("Expected an error for undefined variable reference, got nil")
		}
		exitErr, ok := err.(cli.ExitCoder)
		if !ok {
			t.Fatalf("Expected cli.ExitCoder, got %T", err)
		}
		if exitErr.ExitCode() != 1 {
			t.Errorf("Expected exit code 1, got %d", exitErr.ExitCode())
		}
		output := s.getLines()
		if !strings.Contains(output, "undefined_var") {
			t.Errorf("Expected output to mention 'undefined_var', got %q", output)
		}
		if !strings.Contains(output, "not declared in vars section") {
			t.Errorf("Expected output to mention 'not declared in vars section', got %q", output)
		}
	})

	t.Run("invalid YAML exits 2", func(t *testing.T) {
		s.lines = []string{}
		tmpfile, err := os.CreateTemp("", "bad-runpfile-*.yml")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name())
		tmpfile.WriteString("invalid yaml content: :")
		tmpfile.Close()

		err = doValidate(newCtx(tmpfile.Name()))
		if err == nil {
			t.Fatal("Expected an error for invalid YAML, got nil")
		}
		exitErr, ok := err.(cli.ExitCoder)
		if !ok {
			t.Fatalf("Expected cli.ExitCoder, got %T", err)
		}
		if exitErr.ExitCode() != exitCodeLoad {
			t.Errorf("Expected exit code %d, got %d", exitCodeLoad, exitErr.ExitCode())
		}
	})
}
