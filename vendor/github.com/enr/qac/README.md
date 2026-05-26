# qac

![CI Linux](https://github.com/enr/qac/workflows/CI%20Nix/badge.svg)
![CI Windows](https://github.com/enr/qac/workflows/CI%20Windows/badge.svg)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/enr/qac)](https://pkg.go.dev/github.com/enr/qac)
[![Go Report Card](https://goreportcard.com/badge/github.com/enr/qac)](https://goreportcard.com/report/github.com/enr/qac)

`qac` is a Go library to test _end to end_ command line tools.

A test plan is written in YAML format.

```yaml
preconditions:
  fs:
    - file: ../go.mod
specs:
  cat:
    command:
      working_dir: ../
      cli: cat go.mod
    expectations:
      status:
        equals_to: 0
      output:
        stdout:
          equals_to_file: ../go.mod
```

## Quick start

### Go tests (idiomatic)

```go
import (
  "testing"
  "github.com/enr/qac"
)

func TestExecution(t *testing.T) {
  qac.NewLauncher().ExecuteFileT(t, "plan.yaml")
}
```

`ExecuteFileT` calls `t.Fatalf` on load errors and `t.Errorf` for every spec
failure — no boilerplate needed.

### Go tests (manual report)

```go
func TestExecution(t *testing.T) {
  launcher := qac.NewLauncher()
  report, err := launcher.ExecuteFile("plan.yaml")
  if err != nil {
    t.Fatal(err)
  }
  reporter := qac.NewTestLogsReporter(t)
  reporter.Publish(report)
  for _, err := range report.AllErrors() {
    t.Errorf("error %v", err)
  }
}
```

### Programmatic usage

```go
command := qac.Command{
  Exe: "echo",
  Args: []string{"foo"},
}

stdErrEmpty := true
expectations := qac.Expectations{
  StatusAssertion: qac.StatusAssertion{
    EqualsTo: new(int), // 0
  },
  OutputAssertions: qac.OutputAssertions{
    Stdout: qac.OutputAssertion{EqualsTo: "foo"},
    Stderr: qac.OutputAssertion{IsEmpty: &stdErrEmpty},
  },
}

plan := qac.TestPlan{
  Specs: map[string]qac.Spec{
    "echo": {Command: command, Expectations: expectations},
  },
}

report := qac.NewLauncher().Execute(plan)
for _, block := range report.Blocks() {
  for _, entry := range block.Entries() {
    fmt.Printf(" - %s %s %v\n", entry.Kind().String(), entry.Description(), entry.Errors())
  }
}
```

### Fluent builder API

```go
plan := qac.NewPlan().
  Setup(qac.ShellCmd("mkdir -p /tmp/mytest")).
  Teardown(qac.ShellCmd("rm -rf /tmp/mytest")).
  Spec("write-file", qac.NewSpec().
    Command(qac.ShellCmd("echo hello > /tmp/mytest/out.txt")).
    ExpectStatus(0).
    Build()).
  Spec("read-file", qac.NewSpec().
    Command(qac.Cmd("cat", "/tmp/mytest/out.txt")).
    ExpectStatus(0).
    ExpectStdout(qac.Contains("hello")).
    Build()).
  Build()

qac.NewLauncher().ExecuteT(t, plan)
```

## YAML plan reference

### Top-level structure

```yaml
include:           # merge specs from other plan files
vars:              # plan-level variables
preconditions:     # halt the plan if these fail
setup:             # commands to run before any spec
teardown:          # commands to run after all specs (always runs)
specs:             # the tests
```

### Command fields

Every `command` block (in `specs`, `setup`, and `teardown`) accepts the
following fields.

#### cli vs exe+args

`cli` runs the value through the system shell (`$SHELL -c` on Unix, `cmd /C`
on Windows) — shell features like pipes, redirects, and globs work. `exe` +
`args` starts the process directly without a shell. The two are mutually
exclusive.

```yaml
command:
  cli: cat /etc/hosts | grep localhost   # shell — pipes work

command:
  exe: /usr/bin/grep
  args: [localhost, /etc/hosts]          # direct — no shell
```

#### working_dir

Sets the working directory for the command. Relative paths are resolved from
the directory that contains the YAML file. `~` is expanded to the current
user's home directory.

```yaml
command:
  working_dir: ~/projects/myapp   # ~ expansion
  cli: make test

command:
  working_dir: ../                # relative to the plan file
  cli: go build ./...
```

When `working_dir` is omitted the directory of the plan file is used.

#### env

Injects extra environment variables into the command process without touching
the parent environment. Useful for credentials, feature flags, or locale
settings.

```yaml
command:
  cli: ./myapp --config config.yaml
  env:
    APP_ENV: test
    LOG_LEVEL: debug
```

#### timeout

Maximum wall-clock time to wait for the command. Accepts any duration string
understood by Go's `time.ParseDuration` (e.g. `"30s"`, `"1m30s"`, `"500ms"`).
When the timeout expires the spec is recorded as `TIMEOUT` and the command's
output is not checked.

```yaml
command:
  cli: curl -sf http://localhost:8080/health
  timeout: 10s
```

#### stdin / stdin_file

Pipe a string literal or a file's contents to the command's standard input.
`stdin` and `stdin_file` are mutually exclusive.

```yaml
command:
  exe: cat
  stdin: "hello world"        # inline string

command:
  cli: wc -l
  stdin_file: ./testdata/input.txt   # contents of a file
```

#### ext (platform-specific extension)

`ext` appends a platform-specific suffix to `exe` at runtime, based on
`runtime.GOOS`. Use it to write a single plan that runs on both Unix and
Windows without duplicating specs.

```yaml
specs:
  run-tool:
    command:
      exe: ./bin/mytool
      ext:
        windows: .exe   # becomes ./bin/mytool.exe on Windows
        unix: ""        # no suffix on Unix (can be omitted)
      args: [--version]
    expectations:
      status:
        equals_to: 0
```

`ext` is also accepted on file and filesystem assertions to make path checks
cross-platform:

```yaml
expectations:
  fs:
    - file: ./bin/mytool
      ext:
        windows: .exe
```

### Preconditions

Preconditions are checked before the plan (or spec) runs. If any check fails,
the plan stops (or the spec is skipped). Preconditions do not affect teardown.

```yaml
preconditions:
  fs:
    - file: /etc/resolv.conf       # assert file exists
    - directory: ./output          # assert directory exists
    - file: ./lock                 # assert file does NOT exist
      exists: false
    - file: /etc/resolv.conf
      contains_all:                # assert file content
        - nameserver
```

Specs can have their own `preconditions` block with the same syntax.

### Setup and teardown

Commands listed under `setup` run before specs (or before a single spec); those
under `teardown` run after. Teardown always runs, even if setup or specs fail.
Plan-level setup stops the plan on first failure; spec-level setup stops that
spec only.

```yaml
setup:
  - cli: mkdir -p /tmp/testdata

teardown:
  - cli: rm -rf /tmp/testdata

specs:
  write-and-verify:
    setup:
      - cli: sh -c "echo test-data > /tmp/testdata/input.txt"
    teardown:
      - cli: rm -f /tmp/testdata/input.txt
    command:
      cli: cat /tmp/testdata/input.txt
    expectations:
      output:
        stdout:
          equals_to: test-data
```

### Retries

Use `retries` (number of extra attempts) and `retry_delay` (wait between
attempts) to handle flaky tests. Only the outcome of the final attempt is
recorded; intermediate failures emit an info entry.

```yaml
specs:
  flaky-service:
    retries: 3
    retry_delay: 2s
    command:
      cli: curl -sf http://localhost:8080/health
    expectations:
      status:
        equals_to: 0
```

### Skip and skip_if

`skip: true` unconditionally skips a spec. `skip_if` skips conditionally based
on environment variables.

```yaml
specs:
  linux-only:
    skip_if:
      env_set: WINDOWS_CI        # skip when the variable is defined (any value)
    command:
      cli: ls /etc

  integration:
    skip_if:
      env_value:
        RUN_INTEGRATION: "false" # skip when the variable equals this value
    command:
      cli: ./integration-test.sh
```

### Tags

Assign tags to specs to run or exclude subsets at call time.

```yaml
specs:
  unit-check:
    tags: [fast, unit]
    command:
      cli: ./check-unit

  slow-check:
    tags: [slow, integration]
    command:
      cli: ./check-integration
```

Filter at call time:

```go
// Run only specs tagged "fast"
launcher.ExecuteFileT(t, "plan.yaml", qac.WithTags("fast"))

// Skip specs tagged "slow"
launcher.ExecuteFileT(t, "plan.yaml", qac.SkipTags("slow"))
```

### FailFast

Stop after the first failing spec (teardown still runs):

```go
launcher.ExecuteFileT(t, "plan.yaml", qac.FailFast())
```

### Include

Merge specs from another plan file. Included specs that share a name with the
base plan are ignored (base wins). Vars from included files become defaults.

```yaml
include:
  - ../shared/common.yaml

specs:
  my-extra-spec:
    command:
      cli: echo extra
    expectations:
      status:
        equals_to: 0
```

Merge semantics:
- `vars`: included vars are defaults; base vars override.
- `preconditions`: included checks run first.
- `setup`: included commands run first.
- `teardown`: included commands run last.
- `specs`: included specs not already in base are appended.

### Variables

Define plan-level variables under `vars` and reference them with `${name}`.
Environment variables are available as `${env.NAME}`.

```yaml
vars:
  bin: ./myapp
  output_dir: /tmp/mytest

specs:
  run:
    command:
      cli: ${bin} --output ${output_dir}
    expectations:
      status:
        equals_to: 0

  check-env:
    command:
      cli: echo ${env.HOME}
    expectations:
      status:
        equals_to: 0
```

### Duration assertions

`expectations.duration` asserts on the wall-clock execution time of the
command. Both `min` and `max` accept any duration string understood by Go's
`time.ParseDuration` (e.g. `"500ms"`, `"2s"`, `"1m30s"`). Either field can be
omitted.

```yaml
specs:
  fast-query:
    command:
      cli: ./myapp query --cached
    expectations:
      status:
        equals_to: 0
      duration:
        max: 200ms   # must complete within 200 ms

  slow-build:
    command:
      cli: make all
    expectations:
      status:
        equals_to: 0
      duration:
        min: 1s      # sanity-check: something actually ran
        max: 5m      # upper bound for CI timeout budgets
```

Note: `timeout` (under `command`) stops the process when the limit is reached;
`duration.max` (under `expectations`) lets the command finish and then fails
the spec if it took too long.

## Working with the report

`Execute` and `ExecuteFile` return a `*TestExecutionReport`. Three convenience
methods cover the most common test-integration patterns.

### Summary()

Returns a one-line human-readable result string — useful as a test log header:

```go
report, _ := launcher.ExecuteFile("plan.yaml")
t.Log(report.Summary()) // "3/5 specs passed (1 skipped)"
```

### FailedSpecs()

Returns the names of the specs that failed, in execution order. Use it to log
or assert on specific failures:

```go
if failed := report.FailedSpecs(); len(failed) > 0 {
  t.Errorf("failing specs: %v", failed)
}
```

### FailWith(t)

Calls `t.Errorf` for every error in the report, prefixed with the spec phase so
failures are easy to locate in test output. It is the manual-report equivalent
of `ExecuteFileT`:

```go
report, err := launcher.ExecuteFile("plan.yaml")
if err != nil {
  t.Fatal(err)
}
t.Log(report.Summary())
report.FailWith(t)
```

Combining all three with a reporter for full visibility:

```go
report, err := launcher.ExecuteFile("plan.yaml", qac.WithTags("fast"))
if err != nil {
  t.Fatal(err)
}
reporter := qac.NewTestLogsReporter(t)
reporter.Publish(report)
t.Log(report.Summary())
if failed := report.FailedSpecs(); len(failed) > 0 {
  t.Logf("failed specs: %v", failed)
}
report.FailWith(t)
```

## Reporters

A `Reporter` publishes a `*TestExecutionReport` for human consumption. The
interface has a single method:

```go
type Reporter interface {
  Publish(report *TestExecutionReport) error
}
```

Two built-in implementations are provided.

### NewTestLogsReporter(t)

Writes to Go's test log via `t.Logf`. Each spec block is printed as a labelled
line with its duration and status (`OK` / `KO` / `SKIP`); errors and skipped
reasons appear as indented sub-lines. Output is only shown by `go test` when
the test fails or `-v` is used.

```go
func TestCLI(t *testing.T) {
  report, err := qac.NewLauncher().ExecuteFile("plan.yaml")
  if err != nil {
    t.Fatal(err)
  }
  reporter := qac.NewTestLogsReporter(t)
  reporter.Publish(report)
  report.FailWith(t)
}
```

Example output (with `-v` or on failure):

```
[1/3] cat                                    (2ms) OK
[2/3] mkdir                                  (1ms) OK
[3/3] rm                                     (0ms) KO
  | KO status: expected 0, got 1
```

### NewConsoleReporter()

Writes the same format to stdout via `fmt.Printf`. Use it in standalone
programs or scripts that run qac outside of `go test`.

```go
report := qac.NewLauncher().Execute(plan)
qac.NewConsoleReporter().Publish(report)
```

### Custom reporter

Implement the `Reporter` interface to produce any output format you need
(JSON, JUnit XML, structured logs, etc.):

```go
type jsonReporter struct{}

func (r *jsonReporter) Publish(report *qac.TestExecutionReport) error {
  return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
    "summary":      report.Summary(),
    "failed_specs": report.FailedSpecs(),
    "success":      report.Success(),
  })
}
```

## Run options reference

| Option | Description |
|---|---|
| `WithTags(tags...)` | Run only specs that have at least one of the given tags |
| `SkipTags(tags...)` | Skip specs that have at least one of the given tags |
| `FailFast()` | Stop after the first spec failure |

All options work with `Execute`, `ExecuteFile`, `ExecuteT`, `ExecuteFileT`,
`DryRun`, and `ListSpecs`.

## Launcher API reference

| Method | Description |
|---|---|
| `Execute(plan, opts...)` | Run a `TestPlan` built in Go |
| `ExecuteFile(path, opts...)` | Load and run a YAML plan file |
| `ExecuteT(t, plan, opts...)` | Like `Execute`; calls `t.Errorf` on failures |
| `ExecuteFileT(t, path, opts...)` | Like `ExecuteFile`; calls `t.Fatalf` on load errors |
| `DryRun(plan, opts...)` | Validate structure and report what would run, without executing any commands |
| `ListSpecs(plan, opts...)` | Return the names of specs that would run (after tag/skip filtering) |

### DryRun

`DryRun` validates the plan and reports which specs would run or be skipped,
without executing commands or touching the filesystem. Useful to verify tag
filters and configuration before a real run.

```go
report := launcher.DryRun(plan, qac.WithTags("fast"))
for _, block := range report.Blocks() {
  for _, entry := range block.Entries() {
    fmt.Println(entry.Kind(), entry.Description())
  }
}
```

### ListSpecs

`ListSpecs` returns the names of specs that would actually run after applying
options, in execution order.

```go
names := launcher.ListSpecs(plan, qac.WithTags("fast"))
fmt.Println(names) // ["spec-a", "spec-b"]
```

## Error classification

Errors returned by `AllErrors()` or collected via `FailWith` / `ExecuteFileT`
may be `*qac.Error` values that carry an `ErrorKind` for programmatic
classification. Use `errors.As` to inspect them:

```go
import "errors"

for _, err := range report.AllErrors() {
  var qErr *qac.Error
  if errors.As(err, &qErr) {
    switch qErr.Kind {
    case qac.KindAssertionFailure:
      // an expectation was not met (wrong status, unexpected output, …)
    case qac.KindInfrastructure:
      // system-level failure: file I/O, path resolution, working dir missing
    case qac.KindConfiguration:
      // the plan is invalid: unknown YAML field, cli+exe both set, bad duration string
    }
  }
}
```

| Kind | When it occurs |
|---|---|
| `KindAssertionFailure` | A spec's expectation was not met |
| `KindInfrastructure` | File I/O error, unresolvable path, missing working directory |
| `KindConfiguration` | Invalid plan: unknown YAML field, mutually exclusive fields, unparseable duration |

Not every error is a `*qac.Error`; plain `error` values may also appear.
Always guard with `errors.As` before accessing `Kind`.

## License

Apache 2.0 - see LICENSE file.

Copyright 2020-TODAY qac contributors
