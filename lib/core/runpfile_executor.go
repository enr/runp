package core

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/bww/impatient"
	"github.com/pkg/errors"
)

// NewExecutor creates new RunpfileExecutor
func NewExecutor(rf *Runpfile) *RunpfileExecutor {
	return &RunpfileExecutor{
		rf:                  rf,
		LoggerFactory:       createProcessLogger,
		environmentSettings: loadEnvironmentSettings(),
	}
}

// RunpfileExecutor Executor implementation for Runpfile.
type RunpfileExecutor struct {
	rf                  *Runpfile
	LoggerFactory       func(string, int, LoggerConfig) Logger
	longest             int
	environmentSettings *EnvironmentSettings
}

func (e *RunpfileExecutor) longestName() int {
	if e.longest > 0 {
		return e.longest
	}
	ln := 0
	for _, process := range e.rf.Units {
		if len(process.Name) > ln {
			ln = len(process.Name)
		}
	}
	e.longest = ln
	return e.longest
}

// Start call start on all processes.
func (e *RunpfileExecutor) Start() {

	var wg sync.WaitGroup

	var host *HostProcess
	var container *ContainerProcess
	var sshTunnel *SSHTunnelProcess
	for _, unit := range e.rf.Units {
		unit.vars = e.rf.Vars
		unit.secretKey = e.rf.SecretKey
		unit.environmentSettings = e.environmentSettings
		// Reset process to force rebuild with updated vars
		unit.process = nil
		if unit.Host != nil {
			host = unit.Host
			host.vars = unit.vars
			host.secretKey = unit.secretKey
			host.stopTimeout = unit.StopTimeout
			host.environmentSettings = e.environmentSettings
		}
		if unit.Container != nil {
			container = unit.Container
			container.vars = unit.vars
			container.secretKey = unit.secretKey
			container.stopTimeout = unit.StopTimeout
			container.environmentSettings = e.environmentSettings
		}
		if unit.SSHTunnel != nil {
			sshTunnel = unit.SSHTunnel
			sshTunnel.vars = unit.vars
			sshTunnel.secretKey = unit.secretKey
			sshTunnel.stopTimeout = unit.StopTimeout
			sshTunnel.environmentSettings = e.environmentSettings
		}
	}

	skipped := []string{}
	var pr PreconditionVerifyResult
	for _, unit := range e.rf.Units {
		if unit.Host != nil {
			host = unit.Host
			pr = host.VerifyPreconditions()
			if pr.Vote != Proceed {
				skipped = append(skipped, unit.Name)
				ui.WriteLinef("Preconditions not satisfied (%v): %v", pr.Vote, pr.Reasons)
				continue
			}
		}
		if unit.Container != nil {
			container = unit.Container
			pr = container.VerifyPreconditions()
			if pr.Vote != Proceed {
				skipped = append(skipped, unit.Name)
				ui.WriteLinef("Preconditions not satisfied (%s): %v", pr.Vote, pr.Reasons)
				continue
			}
		}
		if unit.SSHTunnel != nil {
			sshTunnel = unit.SSHTunnel
			pr = sshTunnel.VerifyPreconditions()
			if pr.Vote != Proceed {
				skipped = append(skipped, unit.Name)
				ui.WriteLinef("Preconditions not satisfied (%s): %v", pr.Vote, pr.Reasons)
				continue
			}
		}
	}

	if len(skipped) > 0 {
		ui.WriteLinef("Units skipped due to unsatisfied preconditions: %v", skipped)
	}

	for _, unit := range e.rf.Units {
		if sliceContains(skipped, unit.Name) {
			ui.WriteLinef("Skipped unit %s", unit.Name)
			continue
		}
		wg.Add(1)
		go e.startUnit(unit, &wg)
	}

	wg.Wait()
}

func await(duration time.Duration, resources []string) error {
	ui.WriteLinef(`Awaiting resources for %s: %v resources %v`, duration, len(resources), resources)
	if len(resources) < 1 {
		ui.WriteLinef(`No resources specified, sleeping for %s`, duration)
		time.Sleep(duration)
		return nil
	}
	ui.WriteLinef(`Awaiting %v resources for %s: %v`, len(resources), duration, resources)
	return impatient.Await(context.Background(), resources, duration)
}

func (e *RunpfileExecutor) startUnit(unit *RunpUnit, wg *sync.WaitGroup) {
	logger := e.LoggerFactory(unit.Name, e.longestName(), processLoggerConfiguration)
	// unit.SetEnvironmentSettings(e.environmentSettings)
	process := unit.Process()
	logger.WriteLinef("Starting %s using working dir %s", unit.Name, process.Dir())
	appContext := GetApplicationContext()
	appContext.RegisterRunningProcess(process)
	cmd, err := process.StartCommand()
	if err != nil {
		logger.WriteLinef("Failed to build command for unit %s: %v", unit.Name, err)
		appContext.AddReport(err.Error())
		appContext.RemoveRunningProcess(process)
		wg.Done()
		return
	}

	if process.ShouldWait() {
		start := time.Now()
		resources := []string{}
		if process.AwaitResource() != "" {
			resources = append(resources, process.AwaitResource())
		}
		duration, err := time.ParseDuration(process.AwaitTimeout())
		if err != nil {
			logger.WriteLinef("Invalid duration format '%s': %v", process.AwaitTimeout(), err)
			appContext.AddReport(err.Error())
			appContext.RemoveRunningProcess(process)
			wg.Done()
			return
		}

		err = await(duration, resources)
		if err != nil {
			if err == impatient.ErrTimeout {
				logger.WriteLinef("Timeout error while awaiting resources: %v", err)
			} else {
				logger.WriteLinef("Error occurred while awaiting resources: %v", err)
			}
			ctx := fmt.Sprintf("command %s await %s %s", process.ID(), process.AwaitResource(), process.AwaitTimeout())
			logger.WriteLinef("%+v", errors.Wrap(err, ctx))
			appContext.AddReport(err.Error())
			appContext.RemoveRunningProcess(process)
			wg.Done()
			return
		}
		t1 := time.Now()
		diff := t1.Sub(start)
		logger.WriteLinef("Starting %s at %v (waited %v for %s)", process.ID(), time.Now(), diff, process.AwaitResource())
	}
	logger.WriteLinef("%s command %v", process.ID(), cmd)

	startable, err := process.IsStartable()
	if err != nil {
		logger.WriteLinef("Failed to verify startability for %s: %+v", process.ID(), errors.Wrap(err, "is startable"))
		appContext.RemoveRunningProcess(process)
		wg.Done()
		return
	}
	if !startable {
		logger.WriteLinef("Process %s cannot be started", process.ID())
		appContext.RemoveRunningProcess(process)
		wg.Done()
		return
	}
	r, w, _ := os.Pipe()
	cmd.Stdout(w)
	cmd.Stderr(w)

	// start process and manage errors such as "command not found"
	var pwg sync.WaitGroup
	pwg.Add(1)
	err = cmd.Start()

	if err != nil {
		w.Close()
		ctx := fmt.Sprintf("start process %s", unit.Name)
		logger.WriteLinef("Failed to start process %s: %+v", unit.Name, errors.Wrap(err, ctx))
		appContext.RemoveRunningProcess(process)
		pwg.Done()
		wg.Done()
		return
	}
	logger.Debugf("Process %s successfully started", process.ID())

	w.Close()
	exit := make(chan error, 2)
	go func() {
		exit <- cmd.Wait()
		logger.WriteLinef(`Finished %s %s`, process.ID(), cmd)
	}()

	go func() {
		e := <-exit
		if e != nil {
			// Check if it's an ExitError from signal termination (normal graceful shutdown)
			if exitErr, ok := e.(*exec.ExitError); ok {
				// Check if the process was terminated by a signal (SIGTERM, SIGINT, etc.)
				// This is expected during graceful shutdown and shouldn't be treated as an error
				errMsg := e.Error()
				// Check if error message indicates signal termination
				if errMsg == "signal: terminated" || errMsg == "signal: interrupt" ||
					errMsg == "signal: killed" {
					// Process was terminated by signal (graceful shutdown)
					logger.Debugf("Process %s terminated by signal (graceful shutdown): %s", process.ID(), errMsg)
					appContext.RemoveRunningProcess(process)
					pwg.Done()
					return
				}
				// Also check exit code: 128+signal number indicates signal termination on Unix
				exitCode := exitErr.ExitCode()
				if exitCode == 128+int(syscall.SIGTERM) || exitCode == 128+int(syscall.SIGINT) ||
					exitCode == 128+int(syscall.SIGKILL) {
					// Process was terminated by signal (graceful shutdown)
					logger.Debugf("Process %s terminated by signal (graceful shutdown), exit code: %d", process.ID(), exitCode)
					appContext.RemoveRunningProcess(process)
					pwg.Done()
					return
				}
			}

			switch e.(type) {
			case *os.SyscallError:
				logger.WriteLinef("System call error: %s", e.Error())
			default:
				logger.WriteLinef("Unexpected error type encountered: %T", e)
			}
			ctx := fmt.Sprintf("run process %s", process.ID())
			logger.WriteLinef("Error occurred while running process %s: %+v", process.ID(), errors.Wrap(e, ctx))
			var b bytes.Buffer
			fmt.Fprintf(&b, "Error type %T occurred in process %s: %s", e, process.ID(), e.Error())
			logger.Write(b.Bytes())
			//}
			appContext.AddReport(e.Error())
		} else {
			logger.WriteLinef(`Process %s completed successfully`, process.ID())
		}
		appContext.RemoveRunningProcess(process)
		pwg.Done()
	}()

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		logger.Write(scanner.Bytes())
	}
	if err = scanner.Err(); err != nil {
		logger.WriteLinef("Failed to read output from process %s: %s", process.ID(), err)
	}
	// wait for all goroutines
	pwg.Wait()
	wg.Done()
}
