package core

import (
	"fmt"
	"io/ioutil"
	mr "math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
)

func captureOutput(f func(), t *testing.T) string {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	stderr := os.Stderr
	defer func() {
		os.Stdout = stdout
		os.Stderr = stderr
	}()
	os.Stdout = w
	os.Stderr = w
	f()
	err = w.Close()
	if err != nil {
		t.Fatal(err)
	}
	out, _ := ioutil.ReadAll(r)
	return string(out)
}

// remove "\r", "\r\n" and "\n" from the string if at start or end
func noCR(s string) string {
	s = strings.TrimLeft(s, "\r\n")
	s = strings.TrimLeft(s, "\r")
	s = strings.TrimLeft(s, "\n")
	s = strings.TrimRight(s, "\r\n")
	s = strings.TrimRight(s, "\n")
	s = strings.TrimRight(s, "\r")
	return s
}

func TestCRLF(t *testing.T) {
	message := `hello world`
	longest := 7
	format := fmt.Sprintf(`%%%ds | `, longest)
	sut := CreateMainLogger(`name`, longest, format, true, false)

	expected := fmt.Sprintf("\r   name | %s\r\n", message)

	out := captureOutput(func() {
		sut.WriteLine(message)
	}, t)
	if out != expected {
		t.Logf("Expected : %q", expected)
		t.Logf("Got      : %q", out)
		t.Errorf("Error CRLF WriteLine")
	}
}

func TestWriteLinef(t *testing.T) {

	message := `hello world`
	expected := fmt.Sprintf("   name | %s\n", message)
	longest := 7
	format := fmt.Sprintf(`%%%ds | `, longest)
	sut := &clogger{idx: ci, proc: `name`, longest: longest, format: format, debug: true, colors: false}

	world := `world`

	var written int
	var err error
	out := captureOutput(func() {
		written, err = sut.WriteLinef(`hello %s`, world)
	}, t)
	if noCR(out) != noCR(expected) {
		t.Errorf("Expected output '%s', got '%s'", expected, out)
	}
	if written != len(message) {
		t.Errorf("Expected written '%d', got '%d'", len(message), written)
	}
	if err != nil {
		t.Errorf("Expected no error but got %v", err)
	}
}

func salt() string {
	i := mr.Int()
	return strconv.Itoa(i)
}
func TestWrite(t *testing.T) {

	message := fmt.Sprintf(`test-%s`, salt())
	expected := fmt.Sprintf("   test | %s\n", message)
	longest := 7
	format := fmt.Sprintf(`%%%ds | `, longest)
	sut := &clogger{idx: ci, proc: `test`, longest: longest, format: format, debug: true, colors: false}

	var written int
	var err error
	out := captureOutput(func() {
		written, err = sut.Write([]byte(message))
	}, t)
	if noCR(out) != noCR(expected) {
		t.Errorf("Expected output '%s', got '%s'", expected, out)
	}
	if written != len(message) {
		t.Errorf("Expected written '%d', got '%d'", len(message), written)
	}
	if err != nil {
		t.Errorf("Expected no error but got %v", err)
	}
}

func TestLogDebug(t *testing.T) {

	longest := 7
	format := fmt.Sprintf(`%%%ds | `, longest)
	sut := &clogger{idx: ci, proc: `name`, longest: longest, format: format, debug: false, colors: false}

	world := `world`

	var written int
	var err error
	out := captureOutput(func() {
		written, err = sut.Debugf(`hello %s`, world)
	}, t)
	if out != `` {
		t.Errorf("Expected no output, got '%s'", out)
	}
	if written != 0 {
		t.Errorf("Expected written '0', got '%d'", written)
	}
	if err != nil {
		t.Errorf("Expected no error but got %v", err)
	}
}

func TestDebug(t *testing.T) {
	longest := 7
	format := fmt.Sprintf(`%%%ds | `, longest)

	// Test with debug enabled.
	sut := &clogger{idx: ci, proc: `test`, longest: longest, format: format, debug: true, colors: false}
	message := `debug message`

	var written int
	var err error
	expected := fmt.Sprintf("   test | %s\n", message)
	out := captureOutput(func() {
		written, err = sut.Debug(message)
	}, t)

	if noCR(out) != noCR(expected) {
		t.Errorf("Expected output '%s', got '%s'", expected, out)
	}
	if written != len(message) {
		t.Errorf("Expected written '%d', got '%d'", len(message), written)
	}
	if err != nil {
		t.Errorf("Expected no error but got %v", err)
	}

	// Test with debug disabled.
	sut2 := &clogger{idx: ci, proc: `test2`, longest: longest, format: format, debug: false, colors: false}
	out2 := captureOutput(func() {
		written, err = sut2.Debug(message)
	}, t)

	if out2 != `` {
		t.Errorf("Expected no output when debug is false, got '%s'", out2)
	}
	if written != 0 {
		t.Errorf("Expected written '0' when debug is false, got '%d'", written)
	}
	if err != nil {
		t.Errorf("Expected no error but got %v", err)
	}
}

func TestResetColor(t *testing.T) {
	// ResetColor calls an external function, but we can verify that it doesn't cause a panic.
	// We can't directly check the output because it modifies terminal colors.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ResetColor caused a panic: %v", r)
		}
	}()

	ResetColor()
	// If we get here without a panic, the test has passed.
}

func TestCreateProcessLogger(t *testing.T) {
	// Test createProcessLogger.
	logger := createProcessLogger("testproc", 10, LoggerConfig{
		Debug: true,
		Color: false,
	})

	if logger == nil {
		t.Error("createProcessLogger should return a non-nil logger")
	}

	// Verify that the logger works.
	message := "test message"
	expected := fmt.Sprintf("  testproc | %s\n", message)
	out := captureOutput(func() {
		logger.WriteLine(message)
	}, t)

	if noCR(out) != noCR(expected) {
		t.Errorf("Expected output '%s', got '%s'", expected, out)
	}

	// Test with an empty proc.
	logger2 := createProcessLogger("", 5, LoggerConfig{
		Debug: false,
		Color: true,
	})

	if logger2 == nil {
		t.Error("createProcessLogger should return a non-nil logger even with empty proc")
	}

	// Verify that nothing is written when debug is false.
	out2 := captureOutput(func() {
		logger2.Debug("should not appear")
	}, t)

	if out2 != "" {
		t.Errorf("Expected no output when debug is false, got '%s'", out2)
	}
}
