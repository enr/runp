package core

import (
	"testing"
)

// Mirrors SSHTunnelProcess struct to help write assertions
type expectedSSHTunnelProcess struct {
	// servers
	local  string
	jump   string
	target string
	// auth
	authIdentityFile    string
	authSecret          string
	authEncryptedSecret string
	// generic properties
	workingDir    string
	env           map[string]string
	vars          map[string]string
	awaitTimeout  string
	awaitResource string
}

func TestParsingSSHTunnelProcess(t *testing.T) {

	ConfigureUI(testLogger, LoggerConfig{
		Debug: true,
		Color: false,
	})

	spec := `
local:
  port: 9000
jump:
  host: 192.168.0.4
  port: 2200

---`

	cp := &SSHTunnelProcess{}
	err := unmarshalStrict([]byte(spec), &cp)

	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}

	expected := expectedSSHTunnelProcess{
		local: `localhost:9000`,
		jump:  `192.168.0.4:2200`,
	}

	assertSSHTunnelProcess(cp, expected, t)
}

func assertSSHTunnelProcess(cp *SSHTunnelProcess, expected expectedSSHTunnelProcess, t *testing.T) {
	if expected.local != "" && expected.local != cp.Local.String() {
		t.Errorf(`local host expected:%s got:%s`, expected.local, cp.Local.String())
	}
	if expected.jump != "" && expected.jump != cp.Jump.String() {
		t.Errorf(`jump host expected:%s got:%s`, expected.jump, cp.Jump.String())
	}
	if expected.target != "" && expected.target != cp.Target.String() {
		t.Errorf(`target host expected:%s got:%s`, expected.target, cp.Target.String())
	}
}
