package core

import (
	"bytes"
	"fmt"
	"io"
	mr "math/rand"
	"os"
	"strconv"
	"sync"
	"testing"
)

func collectLogOutput(f func()) string {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	stdout := os.Stdout
	stderr := os.Stderr
	defer func() {
		os.Stdout = stdout
		os.Stderr = stderr
	}()
	os.Stdout = w
	os.Stderr = w

	out := make(chan string)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		var buf bytes.Buffer
		wg.Done()
		io.Copy(&buf, r)
		out <- buf.String()
	}()
	wg.Wait()
	f()
	w.Close()
	return <-out
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
	out := collectLogOutput(func() {
		written, err = sut.WriteLinef(`hello %s`, world)
	})
	if out != expected {
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
	out := collectLogOutput(func() {
		written, err = sut.Write([]byte(message))
	})
	if out != expected {
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
	out := collectLogOutput(func() {
		written, err = sut.Debugf(`hello %s`, world)
	})
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
