// +build windows

package core

func defaultShell() Shell {
	return Shell{
		Path: "cmd",
		Args: []string{
			"/C",
		},
	}
}
