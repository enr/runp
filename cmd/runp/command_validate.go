package main

import (
	"github.com/enr/runp/lib/core"
	"github.com/urfave/cli/v2"
)

func doValidate(c *cli.Context) error {
	f := resolveRunpfileArg(c)
	localPath, displayPath, cleanup, err := fetchRunpfilePath(f, resolveChecksumArg(c))
	defer cleanup()
	if err != nil {
		if isURL(f) {
			return exitErrorf(exitCodeLoad, "Failed to fetch Runpfile: %s", err)
		}
		return runpfileNotFoundError(localPath)
	}

	runpfile, err := core.LoadRunpfileForValidation(localPath)
	if err != nil {
		return exitErrorf(exitCodeLoad, "Failed to load Runpfile %s: %s", displayPath, err.Error())
	}
	ui.WriteLinef("Loaded: %s", displayPath)

	result := core.ValidateRunpfile(runpfile)

	if result.Valid() {
		ui.WriteLinef("valid   %s", displayPath)
		for _, w := range result.Warnings {
			ui.WriteLinef("  warning: %s", w)
		}
		return nil
	}

	ui.WriteLinef("invalid %s", displayPath)
	for _, e := range result.Errors {
		ui.WriteLinef("  - %s", e.Error())
	}
	if len(result.Warnings) > 0 {
		ui.WriteLinef("  warnings:")
		for _, w := range result.Warnings {
			ui.WriteLinef("    - %s", w)
		}
	}
	return cli.NewExitError("", 1)
}
