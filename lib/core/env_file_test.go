package core

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// unsetAfter unsets the given environment variables at the end of the test:
// the Runpfile-level env_file sets them in the environment of runp.
func unsetAfter(t *testing.T, names ...string) {
	t.Helper()
	for _, n := range names {
		os.Unsetenv(n)
	}
	t.Cleanup(func() {
		for _, n := range names {
			os.Unsetenv(n)
		}
	})
}

func TestParseDotenv(t *testing.T) {
	content := "\xef\xbb\xbf# comment\n" +
		"\n" +
		"A=plain\r\n" +
		"export B = spaced value  \n" +
		"C=with # comment\n" +
		"D=no#comment\n" +
		"E='single $HOME \\n'\n" +
		"F=\"double \\\"q\\\" \\n end\" # trailing\n" +
		"G=\n" +
		"H=\"multi\nline\"\n" +
		"I=$NOT_EXPANDED\n" +
		"J=a=b\n"
	got, err := parseDotenv([]byte(content))
	if err != nil {
		t.Fatalf("parseDotenv: %v", err)
	}
	want := map[string]string{
		"A": "plain",
		"B": "spaced value",
		"C": "with",
		"D": "no#comment",
		"E": "single $HOME \\n",
		"F": "double \"q\" \n end",
		"G": "",
		"H": "multi\nline",
		"I": "$NOT_EXPANDED",
		"J": "a=b",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseDotenv()\n got  %q\n want %q", got, want)
	}
}

func TestParseDotenvErrorsDoNotLeakValues(t *testing.T) {
	tests := []struct {
		content string
		msg     string
	}{
		{"OK=1\nsecret-without-equals\n", "line 2: expected NAME=value"},
		{"1BAD=s3cr3t\n", `line 1: invalid variable name "1BAD"`},
		{"A=\"s3cr3t\n", "line 1: unterminated quoted value"},
		{"A='s3cr3t' trailing\n", "line 1: unexpected characters after the closing quote"},
	}
	for _, tt := range tests {
		_, err := parseDotenv([]byte(tt.content))
		if err == nil {
			t.Errorf("parseDotenv(%q): expected error", tt.content)
			continue
		}
		if err.Error() != tt.msg {
			t.Errorf("parseDotenv(%q) error = %q, want %q", tt.content, err, tt.msg)
		}
		if strings.Contains(err.Error(), "s3cr3t") {
			t.Errorf("parseDotenv(%q) error leaks the value: %v", tt.content, err)
		}
	}
}

func TestEnvFilesUnmarshal(t *testing.T) {
	no := false
	tests := []struct {
		yaml string
		want EnvFiles
	}{
		{"env_file: a.env\n", EnvFiles{{Path: "a.env"}}},
		{"env_file: [a.env, b.env]\n", EnvFiles{{Path: "a.env"}, {Path: "b.env"}}},
		{"env_file: {path: a.env, required: false}\n", EnvFiles{{Path: "a.env", Required: &no}}},
		{"env_file:\n  - a.env\n  - path: b.env\n    required: false\n", EnvFiles{{Path: "a.env"}, {Path: "b.env", Required: &no}}},
		{"env_file:\n", nil},
	}
	for _, tt := range tests {
		var out struct {
			EnvFile EnvFiles `yaml:"env_file"`
		}
		if err := unmarshalStrict([]byte(tt.yaml), &out); err != nil {
			t.Errorf("%q: %v", tt.yaml, err)
			continue
		}
		if !reflect.DeepEqual(out.EnvFile, tt.want) {
			t.Errorf("%q: got %+v, want %+v", tt.yaml, out.EnvFile, tt.want)
		}
	}
	for _, bad := range []string{
		"env_file: {path: a.env, optional: true}\n",
		"env_file: {required: false}\n",
		"env_file: 1\n",
		"env_file: [[a.env]]\n",
	} {
		var out struct {
			EnvFile EnvFiles `yaml:"env_file"`
		}
		if err := unmarshalStrict([]byte(bad), &out); err == nil {
			t.Errorf("%q: expected error", bad)
		}
	}
}

func TestLoadEnvFilesOrderAndOptional(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{})
	dir := t.TempDir()
	writeTestFile(t, dir, "a.env", "A=1\nB=1\n")
	writeTestFile(t, dir, "sub/b.env", "B=2\n")
	no := false
	got, err := loadEnvFiles(EnvFiles{
		{Path: "a.env"},
		{Path: "{{vars d}}/b.env"},
		{Path: "missing.env", Required: &no},
	}, dir, newCliPreprocessor(map[string]string{"d": "sub"}))
	if err != nil {
		t.Fatalf("loadEnvFiles: %v", err)
	}
	if want := map[string]string{"A": "1", "B": "2"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	_, err = loadEnvFiles(EnvFiles{{Path: "missing.env"}}, dir, nil)
	if err == nil || !strings.Contains(err.Error(), "env_file not found") {
		t.Errorf("missing required file: got error %v", err)
	}
}

func TestHostEnvironmentWithEnvFile(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{})
	dir := t.TempDir()
	writeTestFile(t, dir, "a.env", "DB_PASSWORD=s3cr3t\nSHARED=from-file\nONLY_FILE=x\n")
	writeTestFile(t, dir, "b.env", "SHARED=from-override\n")
	p := &HostProcess{
		Env: map[string]string{
			"SHARED": "from-env-block",
			"URL":    "postgres://u:${DB_PASSWORD}@db",
		},
		envFiles:    EnvFiles{{Path: "a.env"}, {Path: "b.env"}},
		envFileRoot: dir,
	}
	p.SetID("backend")
	env, err := p.environment()
	if err != nil {
		t.Fatalf("environment: %v", err)
	}
	sort.Strings(env)
	want := []string{
		"DB_PASSWORD=s3cr3t",
		"ONLY_FILE=x",
		"SHARED=from-env-block",
		"URL=postgres://u:s3cr3t@db",
	}
	if !reflect.DeepEqual(env, want) {
		t.Errorf("environment()\n got  %q\n want %q", env, want)
	}

	p.envFiles = EnvFiles{{Path: "nope.env"}}
	if _, err := p.StartCommand(); err == nil || !strings.Contains(err.Error(), "env_file not found") {
		t.Errorf("StartCommand with missing env_file: got error %v", err)
	}
}

func TestContainerBuildArgsWithEnvFile(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{})
	dir := t.TempDir()
	writeTestFile(t, dir, ".env.local", "DB_PASSWORD=s3cr3t\nPOSTGRES_USER=file-user\n")
	cp := &ContainerProcess{
		Image: "postgres:17",
		Env: map[string]string{
			"POSTGRES_PASSWORD": "${DB_PASSWORD}",
			"POSTGRES_USER":     "block-user",
		},
		envFiles:    EnvFiles{{Path: ".env.local"}},
		envFileRoot: dir,
	}
	cp.SetID("db")
	args, runnerEnv, err := cp.buildArgsAndEnv()
	if err != nil {
		t.Fatalf("buildArgsAndEnv: %v", err)
	}
	want := []string{
		"run", "-t", "--rm", "--name", "runp-db", "--network", "runp-network",
		// values from env_file reach the runner through its environment
		"-e", "DB_PASSWORD",
		"-e", "POSTGRES_PASSWORD=s3cr3t",
		"-e", "POSTGRES_USER=block-user",
		"postgres:17",
	}
	if !reflect.DeepEqual(args, want) {
		t.Errorf("args\n got  %q\n want %q", args, want)
	}
	if wantEnv := []string{"DB_PASSWORD=s3cr3t"}; !reflect.DeepEqual(runnerEnv, wantEnv) {
		t.Errorf("runner env = %q, want %q", runnerEnv, wantEnv)
	}
}

func TestRunpfileLevelEnvFile(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{})
	unsetAfter(t, "RUNP_TEST_EF_ROOT", "RUNP_TEST_EF_SHARED", "RUNP_TEST_EF_INC")
	dir := t.TempDir()
	writeTestFile(t, dir, "root.env", "RUNP_TEST_EF_ROOT=root\nRUNP_TEST_EF_SHARED=from-root\n")
	writeTestFile(t, dir, "inc/inc.env", "RUNP_TEST_EF_INC=inc\nRUNP_TEST_EF_SHARED=from-inc\n")
	writeTestFile(t, dir, "inc/Runpfile", `env_file: inc.env
units:
  other:
    host:
      command: "true"
`)
	rp := writeTestFile(t, dir, "Runpfile", `env_file: root.env
include:
  - inc/Runpfile
units:
  app:
    host:
      command: "true"
      env:
        A: ${RUNP_TEST_EF_ROOT}-${RUNP_TEST_EF_INC}-${RUNP_TEST_EF_SHARED}
`)
	rf, err := LoadRunpfileFromPath(rp)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	env := rf.Units["app"].Host.resolveEnvironment()
	if want := []string{"A=root-inc-from-root"}; !reflect.DeepEqual(env, want) {
		t.Errorf("env = %q, want %q", env, want)
	}
	if res := ValidateRunpfile(rf); !res.Valid() {
		t.Errorf("unexpected validation errors: %v", res.Errors)
	}
}

func TestRunpfileLevelEnvFileMissing(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{})
	dir := t.TempDir()
	rp := writeTestFile(t, dir, "Runpfile", `env_file: missing.env
units:
  app:
    host:
      command: "true"
`)
	if _, err := LoadRunpfileFromPath(rp); err == nil || !strings.Contains(err.Error(), "env_file not found") {
		t.Errorf("load: got error %v", err)
	}
	rf, err := LoadRunpfileForValidation(rp)
	if err != nil {
		t.Fatalf("load for validation: %v", err)
	}
	res := ValidateRunpfile(rf)
	if res.Valid() || !strings.Contains(res.Errors[0].Error(), "env_file not found") {
		t.Errorf("validate: got %v", res.Errors)
	}
}

func TestValidateUnitEnvFiles(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{})
	dir := t.TempDir()
	writeTestFile(t, dir, "bad.env", "OK=s3cr3t\nnot a line\n")
	rp := writeTestFile(t, dir, "Runpfile", `vars:
  envdir: x
units:
  bad:
    env_file: bad.env
    host:
      command: "true"
  missing:
    env_file: missing.env
    container:
      image: alpine
  optional:
    env_file:
      path: missing.env
      required: false
    host:
      command: "true"
  withvars:
    env_file: "{{vars envdir}}/later.env"
    host:
      command: "true"
  tunnel:
    env_file: bad.env
    ssh_tunnel:
      user: u
      local:
        port: 1
      jump:
        host: h
        port: 22
      target:
        host: t
        port: 2
`)
	rf, err := LoadRunpfileForValidation(rp)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	errs := validateEnvFiles(rf)
	var msgs []string
	for _, e := range errs {
		msgs = append(msgs, e.Error())
		if strings.Contains(e.Error(), "s3cr3t") {
			t.Errorf("error leaks a value: %v", e)
		}
	}
	joined := strings.Join(msgs, "\n")
	for _, want := range []string{
		`unit "bad": env_file`,
		"line 2: expected NAME=value",
		`unit "missing": env_file not found`,
		`unit "tunnel": env_file is not supported for ssh_tunnel units`,
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q in:\n%s", want, joined)
		}
	}
	if len(errs) != 3 {
		t.Errorf("got %d errors, want 3:\n%s", len(errs), joined)
	}
	// {{vars}} in a unit env_file path is supported.
	for _, e := range validateVariableRefs(rf) {
		t.Errorf("unexpected var error: %v", e)
	}
}
