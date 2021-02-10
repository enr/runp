// +build darwin freebsd linux netbsd openbsd

package core

func defaultShell() Shell {
	return Shell{
		Path: "bash",
		Args: []string{
			"--noprofile",
			"--norc",
			"-eo",
			"pipefail",
			"-c",
		},
	}
}
