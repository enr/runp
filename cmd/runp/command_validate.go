package main

import (
	"github.com/enr/runp/lib/core"
	"github.com/urfave/cli/v2"
)

func doValidate(c *cli.Context) error {
	runpfilePath, err := core.ResolveRunpfilePath(c.String("f"))
	if err != nil {
		return runpfileNotFoundError(runpfilePath)
	}

	runpfile, err := core.LoadRunpfileFromPath(runpfilePath)
	if err != nil {
		return exitErrorf(2, "Failed to load Runpfile %s: %s", runpfilePath, err.Error())
	}
	ui.WriteLinef("Loaded: %s", runpfilePath)

	result := core.ValidateRunpfile(runpfile)

	if result.Valid() {
		ui.WriteLinef("valid   %s", runpfilePath)
		for _, w := range result.Warnings {
			ui.WriteLinef("  warning: %s", w)
		}
		return nil
	}

	ui.WriteLinef("invalid %s", runpfilePath)
	for _, e := range result.Errors {
		ui.WriteLinef("  - %s", e.Error())
	}
	if len(result.Warnings) > 0 {
		ui.WriteLinef("  warnings:")
		for _, w := range result.Warnings {
			ui.WriteLinef("    - %s", w)
		}
	}
	return cli.NewExitError("Runpfile validation failed", 1)
}
