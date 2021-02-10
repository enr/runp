package core

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/bww/impatient"
	"github.com/pkg/errors"
)

// NewExecutor creates new RunpfileExecutor
func NewExecutor(rf *Runpfile) *RunpfileExecutor {
	return &RunpfileExecutor{
		rf:            rf,
		LoggerFactory: createProcessLogger,
	}
}

// RunpfileExecutor Executor implementation for Runpfile.
type RunpfileExecutor struct {
	rf            *Runpfile
	LoggerFactory func(string, int, LoggerConfig) Logger
	longest       int
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
		if unit.Host != nil {
			host = unit.Host
		}
		if unit.Container != nil {
			container = unit.Container
		}
		if unit.SSHTunnel != nil {
			sshTunnel = unit.SSHTunnel
		}
	}

	var err error
	// preconditions are executed one for each process type
	if host != nil {
		err = host.Preconditions()
		if err != nil {
			ui.WriteLinef("Error in Preconditions for process host: %v", err)
			return
		}
	}
	if container != nil {
		err = container.Preconditions()
		if err != nil {
			ui.WriteLinef("Error in Preconditions for process container: %v", err)
			return
		}
	}
	if sshTunnel != nil {
		err = sshTunnel.Preconditions()
		if err != nil {
			ui.WriteLinef("Error in Preconditions for process SSH tunnel: %v", err)
			return
		}
	}

	for _, unit := range e.rf.Units {
		wg.Add(1)
		go e.startUnit(unit, &wg)
	}

	wg.Wait()
}

func (e *RunpfileExecutor) startUnit(unit *RunpUnit, wg *sync.WaitGroup) {
	logger := e.LoggerFactory(unit.Name, e.longestName(), processLoggerConfiguration)
	process := unit.Process()
	logger.WriteLinef("Starting %s using working dir %s", unit.Name, process.Dir())
	appContext := GetApplicationContext()
	appContext.RegisterRunningProcess(process)
	cmd, err := process.StartCommand()
	if err != nil {
		logger.WriteLinef("Error building command %v", err)
		appContext.AddReport(err.Error())
		wg.Done()
		return
	}

	if process.ShouldWait() {
		start := time.Now()
		resources := []string{
			process.AwaitResource(),
		}
		duration, err := time.ParseDuration(process.AwaitTimeout())
		if err != nil {
			logger.WriteLinef("Error duration %v", err)
			appContext.AddReport(err.Error())
			wg.Done()
			return
		}

		err = impatient.Await(context.Background(), resources, duration)
		if err != nil {
			if err == impatient.ErrTimeout {
				logger.WriteLinef("Error timeout %v", err)
			} else {
				logger.WriteLinef("Error generic in await %v", err)
			}
			ctx := fmt.Sprintf("command %s await %s %s", process.ID(), process.AwaitResource(), process.AwaitTimeout())
			logger.WriteLinef("%+v", errors.Wrap(err, ctx))
			appContext.AddReport(err.Error())
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
		logger.WriteLinef("Error in %s %+v", process.ID(), errors.Wrap(err, "is startable"))
		wg.Done()
		return
	}
	if !startable {
		logger.WriteLinef("Process %s not startable", process.ID())
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
		logger.WriteLinef("Error in %s %+v", cmd, errors.Wrap(err, ctx))
		pwg.Done()
		wg.Done()
		return
	}
	logger.Debugf("Process %s successfully started", process.ID())

	appContext.RegisterRunningProcess(process)

	w.Close()
	exit := make(chan error, 2)
	go func() {
		exit <- cmd.Wait()
		logger.WriteLinef(`Finished %s %s`, process.ID(), cmd)
	}()

	go func() {
		e := <-exit
		if e != nil {
			switch e.(type) {
			case *os.SyscallError:
				logger.WriteLinef("SyscallError %s", e.Error())
			default:
				logger.WriteLinef("Unexpected error %T", e)
			}
			ctx := fmt.Sprintf("run process %s", process.ID())
			logger.WriteLinef("Error in %s %+v", cmd, errors.Wrap(e, ctx))
			var b bytes.Buffer
			fmt.Fprintf(&b, "Error %T in %s: %s", e, cmd, e.Error())
			logger.Write(b.Bytes())
			//}
			appContext.AddReport(e.Error())
		} else {
			logger.WriteLinef(`%s exited without error`, process.ID())
		}
		appContext.RemoveRunningProcess(process)
		pwg.Done()
	}()

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		logger.Write(scanner.Bytes())
	}
	if err = scanner.Err(); err != nil {
		logger.WriteLinef("Error in %s reading output: %s", process.ID(), err)
	}
	// wait for all goroutines
	pwg.Wait()
	wg.Done()
}
