package main

import (
	"os"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/enr/runp/lib/core"
)

func doUp(c *cli.Context) error {
	runpfile, err := loadRunpfile(c.String("f"))
	if err != nil {
		return err
	}
	vars := runpfile.Vars
	userVars := c.StringSlice(`var`)
	if len(vars) == 0 && len(userVars) > 0 {
		return exitErrorf(4, `got var from command line but no var is registered in runpfile`)
	}
	var kv []string
	for _, v := range userVars {
		kv = strings.SplitN(v, `=`, 2)
		vars[kv[0]] = kv[1]
	}

	if len(vars) == 0 {
		vars = make(map[string]string)
	}
	wd, err := os.Getwd()
	if err != nil {
		ui.WriteLinef(`Failed to resolve current working directory: %v`, err)
	}
	vars[`runp_root`] = runpfile.Root
	vars[`runp_workdir`] = wd
	vars[`runp_file_separator`] = string(os.PathSeparator)
	runpfile.Vars = vars

	kev := c.String(`key-env`)
	key := c.String(`key`)
	if kev != "" && key != "" {
		return exitErrorf(3, `key and key-env used: they are mutually exclusive`)
	}
	if kev != "" {
		ev := os.Getenv(kev)
		if ev == "" {
			return exitErrorf(3, `key-env "%s" empty`, kev)
		}
		runpfile.SecretKey = ev
	}
	if key != "" {
		runpfile.SecretKey = key
	}

	ui.Debugf("Up using runpfile root %s", runpfile.Root)
	executor := core.NewExecutor(runpfile)
	executor.Start()
	if err != nil {
		return exitErrorf(3, "Failed to execute Runpfile %s", c.String("f"))
	}
	return nil
}
