# qac

![CI Linux](https://github.com/enr/qac/workflows/CI%20Nix/badge.svg)
![CI Windows](https://github.com/enr/qac/workflows/CI%20Windows/badge.svg)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/enr/qac)](https://pkg.go.dev/github.com/enr/qac)
[![Go Report Card](https://goreportcard.com/badge/github.com/enr/qac)](https://goreportcard.com/report/github.com/enr/qac)

`qac` is a Go library to test end to end command line tools.

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
  for _ei_, err := range report.AllErrors() {
    t.Errorf(`error %v`, err)
  }
}
```

## License

Apache 2.0 - see LICENSE file.

Copyright 2020 qac contributors
