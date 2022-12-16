package main

import (
	"strings"

	"github.com/enr/runp/lib/core"
	"github.com/urfave/cli/v2"
)

func doList(c *cli.Context) error {
	runpfile, err := loadRunpfile(c.String("f"))
	if err != nil {
		return err
	}
	ui.WriteLine("Units:")
	for _, u := range runpfile.Units {
		ui.WriteLinef(listLine(u))
	}

	return nil
}

func listLine(u *core.RunpUnit) string {
	var sb strings.Builder
	sb.WriteString(`- `)
	sb.WriteString(u.Name)
	if u.Kind() != "" {
		sb.WriteString(` (`)
		sb.WriteString(u.Kind())
		sb.WriteString(`)`)
	}
	if u.Description != "" {
		sb.WriteString(`: `)
		sb.WriteString(u.Description)
		sb.WriteString(` `)
	}
	return sb.String()
}
