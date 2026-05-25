package core

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const defaultReadyTimeout = 30 * time.Second

// ReadyCondition describes when a unit is considered ready for its dependents
// to start. At most one of Output, Cmd, or Delay should be set; if multiple
// are set, Output takes priority, then Cmd, then Delay.
type ReadyCondition struct {
	// Delay waits a fixed duration after the process starts, e.g. "5s".
	// The unit is considered ready after the duration elapses.
	Delay string `yaml:"delay"`
	// Output is a substring matched against lines from the process stdout/stderr.
	// The unit is considered ready when a line containing this string appears.
	Output string `yaml:"output"`
	// Cmd is a shell command polled every second. The unit is considered ready
	// when the command exits with code 0.
	Cmd string `yaml:"cmd"`
	// Timeout is the maximum time to wait for readiness (default: 30s).
	// Applies to all condition types.
	Timeout string `yaml:"timeout"`
}

// IsSet reports whether any readiness condition is configured.
func (rc ReadyCondition) IsSet() bool {
	return rc.Delay != "" || rc.Output != "" || rc.Cmd != ""
}

func (rc ReadyCondition) parseTimeout() time.Duration {
	if rc.Timeout != "" {
		d, err := time.ParseDuration(rc.Timeout)
		if err == nil {
			return d
		}
	}
	return defaultReadyTimeout
}

// AwaitReady blocks until the ReadyCondition is satisfied or the timeout is
// reached. outputCh receives lines from the process stdout/stderr and is
// consumed only by the ready.output condition; it may be nil for the other
// condition types.
func AwaitReady(rc ReadyCondition, outputCh <-chan string, logger Logger) error {
	if !rc.IsSet() {
		return nil
	}
	timeout := rc.parseTimeout()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if rc.Output != "" {
		logger.WriteLinef("Waiting for output pattern %q (timeout: %s)...", rc.Output, timeout)
		if err := awaitReadyFromOutput(ctx, rc.Output, outputCh); err != nil {
			return err
		}
		logger.WriteLinef("Ready: output pattern %q matched", rc.Output)
		return nil
	}
	if rc.Cmd != "" {
		logger.WriteLinef("Waiting for command %q to succeed (timeout: %s)...", rc.Cmd, timeout)
		if err := awaitReadyFromCmd(ctx, rc.Cmd); err != nil {
			return err
		}
		logger.WriteLinef("Ready: command %q succeeded", rc.Cmd)
		return nil
	}
	d, err := time.ParseDuration(rc.Delay)
	if err != nil {
		return fmt.Errorf("invalid ready delay %q: %w", rc.Delay, err)
	}
	logger.WriteLinef("Waiting %s before signaling ready...", d)
	if err := awaitReadyFromDelay(ctx, d); err != nil {
		return err
	}
	logger.WriteLinef("Ready: delay %s elapsed", d)
	return nil
}

// awaitReadyFromOutput blocks until a line from outputCh contains pattern, or
// until ctx is cancelled.
func awaitReadyFromOutput(ctx context.Context, pattern string, outputCh <-chan string) error {
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("ready_output: timeout waiting for pattern %q: %w", pattern, ctx.Err())
		case line, ok := <-outputCh:
			if !ok {
				return fmt.Errorf("ready_output: process exited before pattern %q appeared in output", pattern)
			}
			if strings.Contains(line, pattern) {
				return nil
			}
		}
	}
}

// awaitReadyFromCmd polls cmdLine every second until it exits with code 0 or
// ctx is cancelled.
func awaitReadyFromCmd(ctx context.Context, cmdLine string) error {
	shell := defaultShell()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("ready_cmd: timeout waiting for command %q: %w", cmdLine, ctx.Err())
		case <-ticker.C:
			args := append(shell.Args, cmdLine)
			cmd := exec.CommandContext(ctx, shell.Path, args...)
			if err := cmd.Run(); err == nil {
				return nil
			}
		}
	}
}

// awaitReadyFromDelay blocks for delay or until ctx is cancelled.
func awaitReadyFromDelay(ctx context.Context, delay time.Duration) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("ready_delay: context cancelled while waiting %s: %w", delay, ctx.Err())
	case <-time.After(delay):
		return nil
	}
}
