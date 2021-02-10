package core

import (
	"strings"
	"testing"

	yaml "gopkg.in/yaml.v2"
)

// Mirrors ContainerProcess struct to help write assertions
type expectedContainerProcess struct {
	id                string
	image             string
	ports             []string
	skipRm            bool
	volumes           []string
	volumesFrom       []string
	mounts            []string
	shmSize           string
	workingDir        string
	env               map[string]string
	command           string
	vars              map[string]string
	commandLineTokens []string
	awaitTimeout      string
	awaitResource     string
}

func assertContainerProcess(actual *ContainerProcess, expected expectedContainerProcess, t *testing.T) {
	if actual.ID() != expected.id {
		t.Errorf(`Container id, expected %s, got %s`, expected.id, actual.ID())
	}
	if actual.Image != expected.image {
		t.Errorf(`Container image, expected %s, got %s`, expected.image, actual.Image)
	}
	assertSliceEquals(actual.Ports, expected.ports, `Ports`, t)
	if actual.SkipRm != expected.skipRm {
		t.Errorf(`Container skip rm, expected %t, got %t`, expected.skipRm, actual.SkipRm)
	}
	assertSliceEquals(actual.Volumes, expected.volumes, `Volumes`, t)
	assertSliceEquals(actual.VolumesFrom, expected.volumesFrom, `Volumes from`, t)
	assertSliceEquals(actual.Mounts, expected.mounts, `Container mounts`, t)
	if actual.ShmSize != expected.shmSize {
		t.Errorf(`Container shm size, expected %s, got %s`, expected.shmSize, actual.ShmSize)
	}
	if actual.WorkingDir != expected.workingDir {
		t.Errorf(`Container working dir, expected %s, got %s`, expected.workingDir, actual.WorkingDir)
	}
	actualCommand := strings.TrimSpace(actual.Command)
	if actualCommand != expected.command {
		t.Errorf(`Container command, expected %s, got %s`, expected.command, actualCommand)
	}

	if expected.awaitTimeout != "" {
		if actual.Await.Timeout != expected.awaitTimeout {
			t.Errorf(`Container await timeout, expected %s, got %s`, expected.awaitTimeout, actual.Await.Timeout)
		}
		if actual.Await.Resource != expected.awaitResource {
			t.Errorf(`Container await resource, expected %s, got %s`, expected.awaitResource, actual.Await.Resource)
		}
	}

	// if actual.buildCmdLine() not contains all expected.commandLineTokens...
}

// docker run --rm --name fowler --mount type=volume,dst=/library/PoEAA
// --mount type=bind,src=/tmp,dst=/library/DSL alpine:3.12 echo "Fowler collection created."
func TestParsing01(t *testing.T) {

	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})

	spec := `
image: alpine:3.12
skip_rm: true
mounts:
  - "type=volume,dst=/library/PoEAA"
  - "type=bind,src=/tmp,dst=/library/DSL"
command: |
  echo "Fowler collection created."
---`

	cp := &ContainerProcess{}
	err := yaml.UnmarshalStrict([]byte(spec), &cp)

	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}

	expected := expectedContainerProcess{
		image:   `alpine:3.12`,
		skipRm:  true,
		mounts:  []string{`type=bind,src=/tmp,dst=/library/DSL`, `type=volume,dst=/library/PoEAA`},
		command: `echo "Fowler collection created."`,
	}

	assertContainerProcess(cp, expected, t)
}

// docker run --rm --name reader --volumes-from fowler --volumes-from knuth alpine:3.12 ls -l /library/
func TestParsing02(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})
	spec := `
image: alpine:3.12
volumes_from:
  - fowler
  - knuth
command: |
  ls -l /library/
await:
  timeout: 0h0m3s
---`

	cp := &ContainerProcess{}
	err := yaml.UnmarshalStrict([]byte(spec), &cp)

	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}

	expected := expectedContainerProcess{
		image:       `alpine:3.12`,
		skipRm:      false,
		volumesFrom: []string{`fowler`, `knuth`},
		command:     `ls -l /library/`,
	}

	assertContainerProcess(cp, expected, t)
}
