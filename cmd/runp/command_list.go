package main

import (
	"fmt"

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
	if u.Description == "" {
		return fmt.Sprintf(`- %s`, u.Name)
	}
	return fmt.Sprintf(`- %s: %s`, u.Name, u.Description)
}
