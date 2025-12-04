package main

import (
	"flag"
	"fmt"
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

func TestExitError(t *testing.T) {
	s := &stubLogger{}
	ui = s
	message := "test error"
	exitCode := 3
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

	if !strings.Contains(s.getLines(), "Error occurred") {
		t.Errorf("Expected UI to contain 'Error occurred', got '%s'", s.getLines())
	}
}

func TestExitErrorf(t *testing.T) {
	s := &stubLogger{}
	ui = s
	template := "error with value %d"
	value := 42
	expectedMessage := fmt.Sprintf(template, value)
	exitCode := 4

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

	if !strings.Contains(s.getLines(), "Error occurred") {
		t.Errorf("Expected UI to contain 'Error occurred', got '%s'", s.getLines())
	}
}

func TestLoadRunpfile(t *testing.T) {
	s := &stubLogger{}
	ui = s
	core.ConfigureUI(s, core.LoggerConfig{})

	// Success case
	t.Run("success", func(t *testing.T) {
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
		if exitErr.ExitCode() != 2 {
			t.Errorf("Expected exit code 2, got %d", exitErr.ExitCode())
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
		if exitErr.ExitCode() != 2 {
			t.Errorf("Expected exit code 2, got %d", exitErr.ExitCode())
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
		if exitErr.ExitCode() != 2 {
			t.Errorf("Expected exit code 2, got %d", exitErr.ExitCode())
		}
		if !strings.Contains(exitErr.Error(), "Invalid Runpfile") {
			t.Errorf("Expected error message to contain 'Invalid Runpfile', got '%s'", exitErr.Error())
		}
	})
}

func TestListLine(t *testing.T) {
	testCases := []struct {
		name     string
		unit     *core.RunpUnit
		expected string
	}{
		{
			name: "Host process",
			unit: &core.RunpUnit{
				Name:        "test-unit",
				Description: "A test unit",
				Host:        &core.HostProcess{},
			},
			expected: "- test-unit (Host process): A test unit ",
		},
		{
			name: "Container process",
			unit: &core.RunpUnit{
				Name:      "test-container",
				Container: &core.ContainerProcess{Image: "test-image"},
			},
			expected: "- test-container (Container process test-image)",
		},
		{
			name: "SSHTunnel process",
			unit: &core.RunpUnit{
				Name: "test-tunnel",
				SSHTunnel: &core.SSHTunnelProcess{
					Local:  core.Endpoint{Host: "localhost", Port: 8080},
					Jump:   core.Endpoint{Host: "", Port: 0}, // Explicitly set Jump to its default zero value
					Target: core.Endpoint{Host: "remotehost", Port: 80},
				},
			},
			expected: "- test-tunnel (SSH tunnel localhost:8080 -> localhost:0 -> remotehost:80)",
		},
		{
			name:     "No process",
			unit:     &core.RunpUnit{Name: "no-process"},
			expected: "- no-process",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Ensure Kind() has a valid process before calling it.
			// This might be required for the SSHTunnel Kind to return the expected value
			if tc.unit.Host != nil || tc.unit.Container != nil || tc.unit.SSHTunnel != nil {
				tc.unit.Process()
			}
			got := listLine(tc.unit)
			if got != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, got)
			}
		})
	}
}

func TestDoList(t *testing.T) {
	s := &stubLogger{}
	ui = s
	core.ConfigureUI(s, core.LoggerConfig{})

	t.Run("success", func(t *testing.T) {
		s.lines = []string{}
		app := cli.NewApp()
		set := flag.NewFlagSet("test", 0)
		set.String("f", "../../testdata/runpfiles/env.yml", "doc")
		c := cli.NewContext(app, set, nil)

		err := doList(c)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		output := s.getLines()
		if !strings.Contains(output, "Units defined in Runpfile:") {
			t.Errorf("Expected output to contain 'Units defined in Runpfile:', got '%s'", output)
		}
		// Corrected expected string for TestDoList/success
		expectedUnitInfo := "- env-test-unit (Host process): echo a user defined environment variable "
		if !strings.Contains(output, expectedUnitInfo) {
			t.Errorf("Expected output to contain '%s', got '%s'", expectedUnitInfo, output)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		s.lines = []string{}
		app := cli.NewApp()
		set := flag.NewFlagSet("test", 0)
		set.String("f", "non-existent-file.yml", "doc")
		c := cli.NewContext(app, set, nil)

		err := doList(c)
		if err == nil {
			t.Fatal("Expected an error, got nil")
		}
		exitErr, ok := err.(cli.ExitCoder)
		if !ok {
			t.Fatalf("Expected an error implementing cli.ExitCoder, got %T", err)
		}
		if exitErr.ExitCode() != 2 {
			t.Errorf("Expected exit code 2, got %d", exitErr.ExitCode())
		}
	})
}
