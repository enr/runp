package core

import (
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestVarNamesAllowUppercaseAndDigits(t *testing.T) {
	p := newCliPreprocessor(map[string]string{"DB_HOST": "db", "port2": "5432"})
	if got := p.process("{{vars DB_HOST}}:{{vars port2}}"); got != "db:5432" {
		t.Errorf("got %q", got)
	}
}

func TestStartFailsOnUndefinedVariable(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{Debug: false, Color: false})
	rf := &Runpfile{
		Vars: map[string]string{"declared": "x"},
		Units: map[string]*RunpUnit{
			"u": {Name: "u", Host: &HostProcess{CommandLine: "echo {{vars missing}}"}},
		},
	}
	rf.Units["u"].Host.SetID("u")
	err := (&RunpfileExecutor{rf: rf, LoggerFactory: createStubLogger}).Start()
	if err == nil || !strings.Contains(err.Error(), "undefined variable") {
		t.Fatalf("expected undefined variable error, got %v", err)
	}
}

func TestValidateAcceptsImplicitVars(t *testing.T) {
	rf := &Runpfile{Units: map[string]*RunpUnit{
		"u": {Host: &HostProcess{WorkingDir: "{{vars runp_root}}/x", CommandLine: "echo {{vars runp_workdir}}"}},
	}}
	if errs := validateVariableRefs(rf); len(errs) != 0 {
		t.Errorf("implicit vars reported as undeclared: %v", errs)
	}
}

func TestExpandEnvValueDollarEscape(t *testing.T) {
	t.Setenv("RUNP_TEST_EXPAND", "value")
	tests := map[string]string{
		"pa$$word":              "pa$word",
		"$RUNP_TEST_EXPAND":     "value",
		"$$RUNP_TEST_EXPAND":    "$RUNP_TEST_EXPAND",
		"${RUNP_TEST_EXPAND}$$": "value$",
		"plain":                 "plain",
	}
	for in, want := range tests {
		if got := expandEnvValue(in); got != want {
			t.Errorf("expandEnvValue(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidateUnitName(t *testing.T) {
	for _, bad := range []string{"", ".", "..", "../x", `a\b`, "a/b", "a\x00b"} {
		if validateUnitName(bad) == nil {
			t.Errorf("unit name %q should be rejected", bad)
		}
	}
	for _, good := range []string{"web", "api-v2", "db_1", "my.service", "Web Server"} {
		if err := validateUnitName(good); err != nil {
			t.Errorf("unit name %q should be accepted: %v", good, err)
		}
	}
	rf := &Runpfile{Units: map[string]*RunpUnit{"../escape": {Host: &HostProcess{}}}}
	if valid, _ := IsRunpfileValid(rf); valid {
		t.Error("Runpfile with a path-traversal unit name should be invalid")
	}
}

func TestContainerNameFilterIsExact(t *testing.T) {
	re := regexp.MustCompile(strings.TrimPrefix(containerNameFilter("runp-api.v1"), "name="))
	for name, want := range map[string]bool{
		"/runp-api.v1":         true,
		"runp-api.v1":          true,
		"/runp-api.v1-gateway": false,
		"/runp-apiXv1":         false,
		"/other-runp-api.v1":   false,
	} {
		if re.MatchString(name) != want {
			t.Errorf("filter match %q = %v, want %v", name, !want, want)
		}
	}
}

func TestContainerBuildArgsProjectLabel(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{Debug: false, Color: false})
	cp := &ContainerProcess{Image: "alpine", project: "abcd1234"}
	cp.SetID("web")
	args, err := cp.buildArgs()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(args, " "), "--label runp.project=abcd1234") {
		t.Errorf("project label missing: %q", args)
	}
}

// recordingProcess records the time it was stopped.
type recordingProcess struct {
	stubProcess
	mu      *sync.Mutex
	stopped *[]string
}

func (p *recordingProcess) StopCommand() (RunpCommand, error) {
	return &recordingStop{p: p}, nil
}

type recordingStop struct {
	SSHTunnelCommandStopper
	p *recordingProcess
}

func (s *recordingStop) Start() error {
	time.Sleep(10 * time.Millisecond)
	s.p.mu.Lock()
	*s.p.stopped = append(*s.p.stopped, s.p.id)
	s.p.mu.Unlock()
	return nil
}

func TestStopInReverseOrder(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{Debug: false, Color: false})
	var mu sync.Mutex
	var stopped []string
	procs := map[string]RunpProcess{}
	for _, id := range []string{"db", "api", "worker", "stray"} {
		procs[id] = &recordingProcess{stubProcess: stubProcess{id: id}, mu: &mu, stopped: &stopped}
	}
	stopInReverseOrder(procs, [][]string{{"db"}, {"api"}, {"worker"}})
	got := strings.Join(stopped, ",")
	if got != "stray,worker,api,db" {
		t.Errorf("stop order = %s, want stray,worker,api,db", got)
	}
}

func TestContainerStopArgsUseStopTimeout(t *testing.T) {
	got := strings.Join(containerStopArgs("runp-db", 1500*time.Millisecond), " ")
	if got != "stop -t 2 runp-db" {
		t.Errorf("containerStopArgs = %q", got)
	}
}

func TestValidateWarnsAwaitWithoutTimeout(t *testing.T) {
	rf := &Runpfile{Units: map[string]*RunpUnit{
		"u": {Host: &HostProcess{CommandLine: "true", Await: AwaitCondition{Resource: "tcp4://localhost:1/"}}},
	}}
	res := ValidateRunpfile(rf)
	found := false
	for _, w := range res.Warnings {
		if strings.Contains(w, "without await.timeout") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected await warning, got %v", res.Warnings)
	}
}

func TestIncludedVarsAreMerged(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{Debug: false, Color: false})
	rf, err := LoadRunpfileFromPath("../../examples/Runpfile-include.yml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if _, ok := rf.Vars["foo"]; !ok {
		t.Fatalf("vars of included files not merged: %v", rf.Vars)
	}
	if res := ValidateRunpfile(rf); !res.Valid() {
		t.Errorf("Runpfile-include.yml should be valid: %v", res.Errors)
	}
}
