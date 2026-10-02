package core

import (
	"sync"
)

// ApplicationContext is a singleton object containing some global variables.
type ApplicationContext struct {
	sync.Mutex
	runningProcesses map[string]RunpProcess
	report           []string
	shuttingDown     bool
	shutdownOnce     sync.Once
	shutdownCh       chan struct{}
}

// RegisterRunningProcess add process to the list of running ones.
func (c *ApplicationContext) RegisterRunningProcess(proc RunpProcess) {
	c.Lock()
	defer c.Unlock()
	c.runningProcesses[proc.ID()] = proc
}

// RemoveRunningProcess add process.
func (c *ApplicationContext) RemoveRunningProcess(proc RunpProcess) {
	c.Lock()
	defer c.Unlock()
	delete(c.runningProcesses, proc.ID())
}

// GetRunningProcesses returns a snapshot of the running processes.
// The returned map is a copy: processes exiting concurrently remove
// themselves from the context without affecting callers iterating it.
func (c *ApplicationContext) GetRunningProcesses() map[string]RunpProcess {
	c.Lock()
	defer c.Unlock()
	out := make(map[string]RunpProcess, len(c.runningProcesses))
	for id, proc := range c.runningProcesses {
		out[id] = proc
	}
	return out
}

// GetReport returns a copy of all reports.
func (c *ApplicationContext) GetReport() []string {
	c.Lock()
	defer c.Unlock()
	return append([]string{}, c.report...)
}

// StopRunningProcesses marks the application as shutting down and stops all
// running processes concurrently, returning when every stop command is done.
func (c *ApplicationContext) StopRunningProcesses() {
	c.SetShuttingDown()
	processes := c.GetRunningProcesses()
	if len(processes) == 0 {
		ui.Debug("No active processes to terminate")
		return
	}
	ui.Debugf("Active processes detected: %d", len(processes))
	stopProcesses(processes)
}

// stopProcesses stops processes concurrently and waits for all of them.
func stopProcesses(processes map[string]RunpProcess) {
	var wg sync.WaitGroup
	for _, process := range processes {
		wg.Add(1)
		go func(process RunpProcess) {
			defer wg.Done()
			stopRunningProcess(process)
		}(process)
	}
	wg.Wait()
}

func stopRunningProcess(process RunpProcess) {
	ui.WriteLinef("Terminating process: %s", process.ID())
	cmd, err := process.StopCommand()
	if err != nil {
		ui.WriteLinef("Failed to load stop command for process %s: %v", process.ID(), err)
		return
	}
	if cmd == nil {
		return
	}
	// Start() calls Stop() which implements graceful shutdown internally
	if err := cmd.Start(); err != nil {
		ui.WriteLinef("Failed to execute stop command for process %s: %v", process.ID(), err)
		return
	}
	// Wait for the stop command to complete (Stop() already handles timeout internally)
	if err := cmd.Wait(); err != nil {
		ui.WriteLinef("Process %s stopped with error: %v", process.ID(), err)
		return
	}
	ui.Debugf("Process %s stopped successfully", process.ID())
}

// AddReport add a report string to the reports list.
func (c *ApplicationContext) AddReport(message string) {
	c.Lock()
	defer c.Unlock()
	c.report = append(c.report, message)
}

// TriggerShutdown signals all listeners (via ShutdownChan) that a graceful
// shutdown has been requested. Safe to call multiple times.
func (c *ApplicationContext) TriggerShutdown() {
	c.shutdownOnce.Do(func() {
		close(c.shutdownCh)
	})
}

// ShutdownChan returns a channel that is closed when TriggerShutdown is called.
func (c *ApplicationContext) ShutdownChan() <-chan struct{} {
	return c.shutdownCh
}

// SetShuttingDown sets the shutting down flag to true.
func (c *ApplicationContext) SetShuttingDown() {
	c.Lock()
	defer c.Unlock()
	c.shuttingDown = true
}

// IsShuttingDown returns true if the application is shutting down.
func (c *ApplicationContext) IsShuttingDown() bool {
	c.Lock()
	defer c.Unlock()
	return c.shuttingDown
}

var (
	once     sync.Once
	instance *ApplicationContext
)

// GetApplicationContext returns the singleton instance of the application context.
func GetApplicationContext() *ApplicationContext {
	once.Do(func() {
		instance = &ApplicationContext{
			runningProcesses: make(map[string]RunpProcess),
			shutdownCh:       make(chan struct{}),
		}
	})
	return instance
}
