//go:build darwin || freebsd || linux || netbsd || openbsd
// +build darwin freebsd linux netbsd openbsd

package core

import (
	"fmt"
	"strings"
	"testing"
)

func TestEnvSubstitution(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})
	//
	runpfilePath := "../../testdata/runpfiles/env.yml"
	rp, err := LoadRunpfileFromPath(runpfilePath)
	if err != nil {
		t.Errorf("Runppfile %s, load error %v", runpfilePath, err)
	}
	sut := &RunpfileExecutor{
		rf:            rp,
		LoggerFactory: createStubLogger,
	}
	sut.Start()
	for _, line := range testLogger.outputLines() {
		fmt.Println(" | " + line)
	}
	expectedStrings := []string{
		"__FOO=A-random-Value__",
	}

	var present bool
	for _, e := range expectedStrings {
		present = false
		for _, a := range testLogger.outputLines() {
			if strings.Contains(a, e) {
				present = true
			}
		}
		if !present {
			t.Errorf("Expected output '%s' not found in actual output %v\n", e, testLogger.outputLines())
		}
	}
}

func TestVarsSubstitution(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})
	//
	runpfilePath := "../../examples/Runpfile-vars.yml"
	rp, err := LoadRunpfileFromPath(runpfilePath)
	if err != nil {
		t.Errorf("Runppfile %s, load error %v", runpfilePath, err)
	}
	sut := &RunpfileExecutor{
		rf:            rp,
		LoggerFactory: createStubLogger,
	}
	sut.Start()
	for _, line := range testLogger.outputLines() {
		fmt.Println(" | " + line)
	}
	expectedStrings := []string{
		"__FOO_DEFAULT_VALUE__",
	}

	var present bool
	for _, e := range expectedStrings {
		present = false
		for _, a := range testLogger.outputLines() {
			if strings.Contains(a, e) {
				present = true
			}
		}
		if !present {
			t.Errorf("Expected output '%s' not found in actual output %v\n", e, testLogger.outputLines())
		}
	}
}
