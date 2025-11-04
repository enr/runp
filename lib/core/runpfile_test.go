package core

import (
	"testing"
)

func TestRunpUnitKind(t *testing.T) {
	// Test per Container process
	t.Run("Container process", func(t *testing.T) {
		unit := &RunpUnit{
			Container: &ContainerProcess{
				Image: "nginx:latest",
			},
		}

		kind := unit.Kind()
		expected := "Container process nginx:latest"
		if kind != expected {
			t.Errorf("Expected '%s', got '%s'", expected, kind)
		}
	})

	// Test per Host process
	t.Run("Host process", func(t *testing.T) {
		unit := &RunpUnit{
			Host: &HostProcess{
				CommandLine: "echo hello",
			},
		}

		kind := unit.Kind()
		expected := "Host process"
		if kind != expected {
			t.Errorf("Expected '%s', got '%s'", expected, kind)
		}
	})

	// Test per SSHTunnel process
	t.Run("SSH tunnel process", func(t *testing.T) {
		unit := &RunpUnit{
			SSHTunnel: &SSHTunnelProcess{
				Local:  Endpoint{Host: "localhost", Port: 8080},
				Jump:   Endpoint{Host: "jump.example.com", Port: 22},
				Target: Endpoint{Host: "target.example.com", Port: 3306},
			},
		}

		kind := unit.Kind()
		expected := "SSH tunnel localhost:8080 -> jump.example.com:22 -> target.example.com:3306"
		if kind != expected {
			t.Errorf("Expected '%s', got '%s'", expected, kind)
		}
	})

	// Test per SSHTunnel process con host vuoto (usa localhost di default)
	t.Run("SSH tunnel process with empty host", func(t *testing.T) {
		unit := &RunpUnit{
			SSHTunnel: &SSHTunnelProcess{
				Local:  Endpoint{Host: "", Port: 8080},
				Jump:   Endpoint{Host: "jump.example.com", Port: 22},
				Target: Endpoint{Host: "target.example.com", Port: 3306},
			},
		}

		kind := unit.Kind()
		expected := "SSH tunnel localhost:8080 -> jump.example.com:22 -> target.example.com:3306"
		if kind != expected {
			t.Errorf("Expected '%s', got '%s'", expected, kind)
		}
	})

	// Test per unit senza processi (dovrebbe restituire stringa vuota)
	t.Run("Empty unit", func(t *testing.T) {
		unit := &RunpUnit{}

		kind := unit.Kind()
		expected := ""
		if kind != expected {
			t.Errorf("Expected '%s', got '%s'", expected, kind)
		}
	})
}
