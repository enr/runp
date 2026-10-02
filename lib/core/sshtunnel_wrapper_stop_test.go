package core

import (
	"testing"
	"time"
)

// Stop must close the local listener so that Wait returns and the tunnel
// stops accepting connections.
func TestSSHTunnelCommandWrapper_StopEndsWait(t *testing.T) {
	ConfigureUI(testLogger, LoggerConfig{Debug: false, Color: false})
	c := &SSHTunnelCommandWrapper{localAddress: "127.0.0.1:0"}
	done := make(chan error, 1)
	go func() { done <- c.Wait() }()

	deadline := time.Now().Add(2 * time.Second)
	for {
		c.mu.Lock()
		listening := c.listener != nil
		c.mu.Unlock()
		if listening || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := c.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Wait returned %v after Stop", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Wait did not return after Stop")
	}
}
