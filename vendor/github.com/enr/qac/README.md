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

Usage in Go tests:

```go
import (
  "testing"
  "github.com/enr/qac"
)
func TestExecution(t *testing.T) {
  launcher := qac.NewLauncher()
  report := launcher.ExecuteFile(`/path/to/qac.yaml`)
  // Not needed but useful to see what's happening
  reporter := qac.NewTestLogsReporter(t)
  reporter.Publish(report)
  // Fail test if any error is found
  for _, err := range report.AllErrors() {
    t.Errorf(`error %v`, err)
  }
}
```

Programmatic usage:

```go
  // the commmand to test
  command := qac.Command{
    Exe: "echo",
    Args: []string{
      `foo`,
    },
  }

  // expectations about its result
  stdErrEmpty := true
  expectations := qac.Expectations{
    StatusAssertion: qac.StatusAssertion{
      EqualsTo: "0",
    },
    OutputAssertions: qac.OutputAssertions{
      Stdout: qac.OutputAssertion{
        EqualsTo: `foo`,
      },
      Stderr: qac.OutputAssertion{
        IsEmpty: &stdErrEmpty,
      },
    },
  }

  // build the full specs structure
  spec := qac.Spec{
    Command:      command,
    Expectations: expectations,
  }
  specs := make(map[string]qac.Spec)
  specs[`echo`] = spec

  // add specs to test plan
  plan := qac.TestPlan{
    Specs: specs,
  }

  // run the plan
  launcher := qac.NewLauncher()

  // see results
  report := launcher.Execute(plan)
  for _, block := range report.Blocks() {
    for _, entry := range block.Entries() {
      fmt.Printf(" - %s %s %v \n", entry.Kind().String(), entry.Description(), entry.Errors())
    }
  }
```

## License

Apache 2.0 - see LICENSE file.

Copyright 2020-TODAY qac contributors
