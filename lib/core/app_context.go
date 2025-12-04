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

// GetRunningProcesses returns all running processes.
func (c *ApplicationContext) GetRunningProcesses() map[string]RunpProcess {
	return c.runningProcesses
}

// GetReport returns all reports.
func (c *ApplicationContext) GetReport() []string {
	return c.report
}

// AddReport add a report string to the reports list.
func (c *ApplicationContext) AddReport(message string) {
	c.Lock()
	defer c.Unlock()
	c.report = append(c.report, message)
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
			runningProcesses: make(map[string]RunpProcess)}
	})
	return instance
}
