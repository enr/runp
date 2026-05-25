//go:build darwin || freebsd || linux || netbsd || openbsd
// +build darwin freebsd linux netbsd openbsd

package core

import (
	"os"
	"strings"
	"testing"
	"time"
)

// --- ReadyCondition.IsSet ---

func TestReadyCondition_IsSet_Empty(t *testing.T) {
	rc := ReadyCondition{}
	if rc.IsSet() {
		t.Error("empty ReadyCondition should not be set")
	}
}

func TestReadyCondition_IsSet_Delay(t *testing.T) {
	rc := ReadyCondition{Delay: "1s"}
	if !rc.IsSet() {
		t.Error("ReadyCondition with Delay should be set")
	}
}

func TestReadyCondition_IsSet_Output(t *testing.T) {
	rc := ReadyCondition{Output: "ready"}
	if !rc.IsSet() {
		t.Error("ReadyCondition with Output should be set")
	}
}

func TestReadyCondition_IsSet_Cmd(t *testing.T) {
	rc := ReadyCondition{Cmd: "true"}
	if !rc.IsSet() {
		t.Error("ReadyCondition with Cmd should be set")
	}
}

// --- ReadyCondition.parseTimeout ---

func TestReadyCondition_parseTimeout_Default(t *testing.T) {
	rc := ReadyCondition{}
	if rc.parseTimeout() != defaultReadyTimeout {
		t.Errorf("expected default timeout %s, got %s", defaultReadyTimeout, rc.parseTimeout())
	}
}

func TestReadyCondition_parseTimeout_Custom(t *testing.T) {
	rc := ReadyCondition{Timeout: "10s"}
	if rc.parseTimeout() != 10*time.Second {
		t.Errorf("expected 10s, got %s", rc.parseTimeout())
	}
}

func TestReadyCondition_parseTimeout_Invalid(t *testing.T) {
	rc := ReadyCondition{Timeout: "not-a-duration"}
	if rc.parseTimeout() != defaultReadyTimeout {
		t.Errorf("invalid timeout should fall back to default, got %s", rc.parseTimeout())
	}
}

// --- AwaitReady: no condition ---

func TestAwaitReady_NoCondition(t *testing.T) {
	rc := ReadyCondition{}
	if err := AwaitReady(rc, nil, testLogger); err != nil {
		t.Errorf("AwaitReady with no condition should return nil, got %v", err)
	}
}

// --- AwaitReady: ready.delay ---

func TestAwaitReady_Delay_Success(t *testing.T) {
	rc := ReadyCondition{Delay: "50ms", Timeout: "5s"}
	start := time.Now()
	if err := AwaitReady(rc, nil, testLogger); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if time.Since(start) < 40*time.Millisecond {
		t.Errorf("expected at least 40ms elapsed")
	}
}

func TestAwaitReady_Delay_InvalidFormat(t *testing.T) {
	rc := ReadyCondition{Delay: "not-valid", Timeout: "5s"}
	if err := AwaitReady(rc, nil, testLogger); err == nil {
		t.Error("expected error for invalid delay format")
	}
}

func TestAwaitReady_Delay_Timeout(t *testing.T) {
	rc := ReadyCondition{Delay: "10s", Timeout: "50ms"}
	if err := AwaitReady(rc, nil, testLogger); err == nil {
		t.Error("expected timeout error when delay exceeds timeout")
	}
}

// --- AwaitReady: ready.output ---

func TestAwaitReady_Output_PatternFound(t *testing.T) {
	rc := ReadyCondition{Output: "listening", Timeout: "5s"}
	ch := make(chan string, 4)
	ch <- "starting server..."
	ch <- "server listening on :3000"
	ch <- "accepting connections"

	if err := AwaitReady(rc, ch, testLogger); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestAwaitReady_Output_PatternFoundSubstring(t *testing.T) {
	rc := ReadyCondition{Output: "ready", Timeout: "5s"}
	ch := make(chan string, 2)
	ch <- "not yet"
	ch <- "system ready to accept connections"

	if err := AwaitReady(rc, ch, testLogger); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestAwaitReady_Output_ChannelClosedBeforeMatch(t *testing.T) {
	rc := ReadyCondition{Output: "listening", Timeout: "5s"}
	ch := make(chan string, 2)
	ch <- "starting..."
	ch <- "crashed before ready"
	close(ch)

	if err := AwaitReady(rc, ch, testLogger); err == nil {
		t.Error("expected error when channel closed before pattern appeared")
	}
}

func TestAwaitReady_Output_Timeout(t *testing.T) {
	rc := ReadyCondition{Output: "never-appears", Timeout: "50ms"}
	ch := make(chan string) // unbuffered, nobody sends

	if err := AwaitReady(rc, ch, testLogger); err == nil {
		t.Error("expected timeout error")
	}
}

func TestAwaitReady_Output_PatternSentAsync(t *testing.T) {
	rc := ReadyCondition{Output: "go!", Timeout: "5s"}
	ch := make(chan string, 4)

	go func() {
		time.Sleep(30 * time.Millisecond)
		ch <- "almost..."
		time.Sleep(30 * time.Millisecond)
		ch <- "go!"
	}()

	if err := AwaitReady(rc, ch, testLogger); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

// --- AwaitReady: ready.cmd ---

func TestAwaitReady_Cmd_Succeeds(t *testing.T) {
	rc := ReadyCondition{Cmd: "true", Timeout: "5s"}
	if err := AwaitReady(rc, nil, testLogger); err != nil {
		t.Fatalf("expected nil for 'true' command, got %v", err)
	}
}

func TestAwaitReady_Cmd_Timeout(t *testing.T) {
	rc := ReadyCondition{Cmd: "false", Timeout: "150ms"}
	if err := AwaitReady(rc, nil, testLogger); err == nil {
		t.Error("expected timeout error for always-failing command")
	}
}

func TestAwaitReady_Cmd_SucceedsAfterRetry(t *testing.T) {
	tmpFile := t.TempDir() + "/ready.flag"

	go func() {
		time.Sleep(1200 * time.Millisecond)
		os.WriteFile(tmpFile, []byte("ready"), 0644) //nolint:errcheck
	}()

	rc := ReadyCondition{Cmd: "test -f " + tmpFile, Timeout: "10s"}
	if err := AwaitReady(rc, nil, testLogger); err != nil {
		t.Fatalf("expected nil after retry, got %v", err)
	}
}

// --- Priority: Output > Cmd > Delay ---

func TestAwaitReady_Priority_OutputOverCmd(t *testing.T) {
	// Both Output and Cmd are set; Output takes priority.
	rc := ReadyCondition{Output: "hi", Cmd: "false", Timeout: "5s"}
	ch := make(chan string, 1)
	ch <- "hi there"

	if err := AwaitReady(rc, ch, testLogger); err != nil {
		t.Fatalf("Output should take priority over Cmd; got %v", err)
	}
}

func TestAwaitReady_Priority_CmdOverDelay(t *testing.T) {
	// Both Cmd and Delay are set; Cmd takes priority.
	rc := ReadyCondition{Cmd: "true", Delay: "10s", Timeout: "5s"}
	start := time.Now()
	if err := AwaitReady(rc, nil, testLogger); err != nil {
		t.Fatalf("Cmd should take priority over Delay; got %v", err)
	}
	if time.Since(start) > 3*time.Second {
		t.Error("Cmd path should complete quickly, not wait for Delay")
	}
}

// --- YAML parsing ---

func TestReadyCondition_YAMLParsing(t *testing.T) {
	spec := `
name: test
units:
  with-delay:
    ready:
      delay: 5s
      timeout: 60s
    host:
      command: echo hi
  with-output:
    ready:
      output: "Server started"
    host:
      command: echo hi
  with-cmd:
    ready:
      cmd: "curl -sf http://localhost:8080/health"
      timeout: 30s
    host:
      command: echo hi
  no-ready:
    host:
      command: echo hi
`
	rf := &Runpfile{}
	if err := unmarshalStrict([]byte(spec), rf); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	u := rf.Units["with-delay"]
	if u == nil {
		t.Fatal("unit with-delay not found")
	}
	if u.Ready.Delay != "5s" {
		t.Errorf("expected delay=5s, got %q", u.Ready.Delay)
	}
	if u.Ready.Timeout != "60s" {
		t.Errorf("expected timeout=60s, got %q", u.Ready.Timeout)
	}
	if !u.Ready.IsSet() {
		t.Error("with-delay should have IsSet()=true")
	}

	u = rf.Units["with-output"]
	if u == nil {
		t.Fatal("unit with-output not found")
	}
	if u.Ready.Output != "Server started" {
		t.Errorf("expected output=%q, got %q", "Server started", u.Ready.Output)
	}

	u = rf.Units["with-cmd"]
	if u == nil {
		t.Fatal("unit with-cmd not found")
	}
	if u.Ready.Cmd != "curl -sf http://localhost:8080/health" {
		t.Errorf("expected cmd=%q, got %q", "curl -sf http://localhost:8080/health", u.Ready.Cmd)
	}

	u = rf.Units["no-ready"]
	if u == nil {
		t.Fatal("unit no-ready not found")
	}
	if u.Ready.IsSet() {
		t.Error("unit no-ready should have empty ReadyCondition")
	}
}

// --- Executor integration: ready.delay ---

func TestStartUnit_ReadyDelay_AdvancesLayer(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{Debug: false, Color: false})

	// infra has a 100ms ready delay and exits immediately.
	// api depends on infra and should start after infra signals ready
	// (not wait for infra to run forever).
	spec := `
name: ready-delay-test
units:
  infra:
    ready:
      delay: 100ms
      timeout: 5s
    host:
      command: "echo infra-done"
  api:
    depends_on: [infra]
    host:
      command: "echo api-started"
`
	rf := &Runpfile{}
	if err := unmarshalStrict([]byte(spec), rf); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	sut := &RunpfileExecutor{
		rf:            rf,
		LoggerFactory: createStubLogger,
		newPipe:       os.Pipe,
	}

	done := make(chan error, 1)
	go func() { done <- sut.Start() }()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("test timed out — api likely never started")
	}

	found := false
	for _, line := range testLogger.outputLines() {
		if strings.Contains(line, "api-started") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'api-started' in output")
	}
}

// --- Executor integration: ready.output ---

func TestStartUnit_ReadyOutput_AdvancesLayer(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{Debug: false, Color: false})

	// server prints READY_SIGNAL and exits; client depends on server.
	spec := `
name: ready-output-test
units:
  server:
    ready:
      output: "READY_SIGNAL"
      timeout: 5s
    host:
      command: "echo READY_SIGNAL"
  client:
    depends_on: [server]
    host:
      command: "echo client-ran"
`
	rf := &Runpfile{}
	if err := unmarshalStrict([]byte(spec), rf); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	sut := &RunpfileExecutor{
		rf:            rf,
		LoggerFactory: createStubLogger,
		newPipe:       os.Pipe,
	}

	done := make(chan error, 1)
	go func() { done <- sut.Start() }()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("test timed out — client likely never started")
	}

	found := false
	for _, line := range testLogger.outputLines() {
		if strings.Contains(line, "client-ran") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'client-ran' in output")
	}
}

// --- Executor integration: ready.cmd ---

func TestStartUnit_ReadyCmd_AdvancesLayer(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{Debug: false, Color: false})

	flagFile := t.TempDir() + "/ready.flag"

	// service writes a flag file and exits; downstream depends on service.
	// ready.cmd polls for the flag file.
	spec := `
name: ready-cmd-test
units:
  service:
    ready:
      cmd: "test -f ` + flagFile + `"
      timeout: 10s
    host:
      command: "touch ` + flagFile + `"
  downstream:
    depends_on: [service]
    host:
      command: "echo downstream-ran"
`
	rf := &Runpfile{}
	if err := unmarshalStrict([]byte(spec), rf); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	sut := &RunpfileExecutor{
		rf:            rf,
		LoggerFactory: createStubLogger,
		newPipe:       os.Pipe,
	}

	done := make(chan error, 1)
	go func() { done <- sut.Start() }()

	select {
	case <-done:
	case <-time.After(12 * time.Second):
		t.Fatal("test timed out — downstream likely never started")
	}

	found := false
	for _, line := range testLogger.outputLines() {
		if strings.Contains(line, "downstream-ran") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'downstream-ran' in output")
	}
}
