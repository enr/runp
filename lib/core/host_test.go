package core

import (
	"strings"
	"testing"

	yaml "gopkg.in/yaml.v2"
)

type expectedHostProcess struct {
	id            string
	commandLine   string
	executable    string
	args          []string
	shell         Shell
	workingDir    string
	env           map[string]string
	vars          map[string]string
	awaitTimeout  string
	awaitResource string
}

func assertHostProcess(actual *HostProcess, expected expectedHostProcess, t *testing.T) {
	if actual.ID() != expected.id {
		t.Errorf(`Container id, expected %s, got %s`, expected.id, actual.ID())
	}
	assertSliceEquals(actual.Args, expected.args, `Args`, t)
	if actual.WorkingDir != expected.workingDir {
		t.Errorf(`Container working dir, expected %s, got %s`, expected.workingDir, actual.WorkingDir)
	}
	actualCommandLine := strings.TrimSpace(actual.CommandLine)
	if actualCommandLine != expected.commandLine {
		t.Errorf(`Container command line, expected %s, got %s`, expected.commandLine, actualCommandLine)
	}

	if expected.awaitTimeout != "" {
		if actual.Await.Timeout != expected.awaitTimeout {
			t.Errorf(`Container await timeout, expected %s, got %s`, expected.awaitTimeout, actual.Await.Timeout)
		}
	}
	if expected.awaitResource != "" {
		if actual.Await.Resource != expected.awaitResource {
			t.Errorf(`Container await resource, expected %s, got %s`, expected.awaitResource, actual.Await.Resource)
		}
	}

	// if sh, args, err := p.resolveShell()...
}

func TestHostProcess01(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})
	spec := `
command: echo 'Hello!'
shell:
  path: /bin/sh
  args:
    - "-c"
---`

	p := &HostProcess{}
	err := yaml.UnmarshalStrict([]byte(spec), &p)

	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}

	expected := expectedHostProcess{
		commandLine: `echo 'Hello!'`,
	}

	assertHostProcess(p, expected, t)

	// fmt.Printf("ERROR %v \n", err)
	// fmt.Printf("%v \n", p.Executable)

	// sh, args, err := p.resolveShell()
	// fmt.Printf("shell %s \n %q \n %v \n", sh, args, err)
	// cl := p.CommandLine
	// fmt.Println(cl)
}
