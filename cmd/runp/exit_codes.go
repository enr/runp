package main

// Exit codes returned by runp. These are stable and may be used by scripts
// and CI pipelines to distinguish failure categories.
//
//	0  success
//	1  validation failure (runp validate found errors)
//	2  load error   – Runpfile not found or could not be parsed
//	3  argument error – bad or mutually-exclusive CLI flags, missing required arg
//	4  variable error – undeclared or circular variable reference
//	5  execution error – process failed to start, preconditions not met
//	128+N  stopped by signal N (130 for Ctrl-C/SIGINT, 143 for SIGTERM)
const (
	exitCodeLoad = 2 // Runpfile not found or could not be parsed
	exitCodeArg  = 3 // bad arguments: mutually exclusive flags, missing required arg
	exitCodeVar  = 4 // invalid / undeclared variable reference
	exitCodeExec = 5 // execution error: process failed to start, preconditions not met
)
