package main

import (
	"errors"

	"github.com/enr/runp/lib/core"
	"github.com/urfave/cli/v2"
)

func doReload(c *cli.Context) error {
	unitName := c.Args().First()
	if unitName == "" {
		return exitErrorf(3, "Usage: runp reload <unit-name>")
	}

	runpfile, err := prepareRunpfile(c)
	if err != nil {
		return err
	}

	unit, ok := runpfile.Units[unitName]
	if !ok {
		return exitErrorf(3, "unit %q not found in Runpfile", unitName)
	}

	pidDir := ""
	if runpfile.Root != "" {
		if d, e := core.PIDDirForRoot(runpfile.Root); e == nil {
			pidDir = d
		}
	}

	envSettings := core.LoadEnvironmentSettings()

	ui.WriteLinef("Stopping unit %s", unitName)
	stopErr := core.StopUnit(unit, pidDir, envSettings)
	if stopErr != nil {
		if errors.Is(stopErr, core.ErrReloadNotSupported) {
			return exitErrorf(3, "reload not supported: %s", stopErr.Error())
		}
		ui.WriteLinef("Warning: stop returned: %s (proceeding with start)", stopErr.Error())
	}

	ui.WriteLinef("Starting unit %s", unitName)
	executor := core.NewExecutor(runpfile)
	if err := executor.StartSingleUnit(unitName); err != nil {
		return exitErrorf(exitCodeExec, "failed to start unit %q: %s", unitName, err.Error())
	}

	ui.WriteLinef("Unit %s reloaded successfully", unitName)
	return nil
}
