// Package core — executor implementation split across:
//   executor_scheduler.go  – struct, init, topological start ordering
//   executor_lifecycle.go  – stop, timeout, signal handling
//   executor_output.go     – per-process output routing
package core
