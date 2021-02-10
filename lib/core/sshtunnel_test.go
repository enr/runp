// +build darwin freebsd linux netbsd openbsd

package core

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/enr/go-files/files"
)

func Test01(t *testing.T) {

	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})

	sshUser := `test`
	sshSecret := `test`
	sshKeyName := `runp`

	var err error
	sshKey, err := filepath.Abs(fmt.Sprintf(`../../testdata/keys/%s`, sshKeyName))
	if err != nil {
		t.Fatal(err)
	}
	if !files.Exists(sshKey) {
		t.Fatalf(`SSH key not found: %s`, sshKey)
	}

	local := Endpoint{
		Port: 3030,
	}
	jump := Endpoint{
		Host: "localhost",
		Port: 6666,
	}
	target := Endpoint{
		Host: "localhost",
		Port: 8089,
	}

	jumpListener, _, _ := testListener(jump, t)
	defer jumpListener.Close()

	sshConfig, err := sshServerConfig(sshUser, sshSecret, sshKey)
	if err != nil {
		t.Fatal(err)
	}

	startSSHServer(jumpListener, sshConfig, t)

	stubResponse := `test-test`
	https := httpServer(target, stubResponse, t)
	defer https.Close()

	tunnel := &SSHTunnelProcess{
		User: sshUser,
		Auth: Auth{
			Secret: sshSecret,
		},
		Local:  local,
		Jump:   jump,
		Target: target,
	}

	cmd, err := tunnel.StartCommand()
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}

	r, w, _ := os.Pipe()
	defer func() {

		w.Close()
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			t.Log(string(scanner.Bytes()))
		}
		if err := scanner.Err(); err != nil {
			t.Logf("Error in reading output: %s", err)
		}
	}()
	cmd.Stdout(w)
	cmd.Stderr(w)

	err = cmd.Start()
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}

	go func() {
		err = cmd.Wait()
		if err != nil {
			t.Errorf("Unexpected error %v", err)
		}
	}()

	var testClient = &http.Client{
		Timeout: time.Second * 2,
	}

	testURL := fmt.Sprintf(`http://%s`, local.String())
	resp, err := testClient.Get(testURL)
	if err != nil {
		t.Errorf("Error calling local %s: %v", testURL, err)
	}
	defer func() {
		if resp == nil || resp.Body == nil {
			t.Errorf("No response from local %s: %v", testURL, err)
			return
		}
		resp.Body.Close()
	}()
	if resp != nil && resp.Body != nil {
		body, _ := ioutil.ReadAll(resp.Body)
		b := strings.TrimSpace(string(body))
		t.Log(b)
		if b != stubResponse {
			t.Errorf("Response from local, expected: <%s> got <%s>", stubResponse, b)
		}
	}

}
