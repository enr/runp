package main

import (
	"github.com/urfave/cli/v2"
)

func doList(c *cli.Context) error {
	runpfile, err := loadRunpfile(c.String("f"))
	if err != nil {
		return err
	}
	ui.WriteLine("Units:")
	for _, u := range runpfile.Units {
		ui.WriteLinef(`- %s: %s`, u.Name, u.Description)
	}

	return nil
}
