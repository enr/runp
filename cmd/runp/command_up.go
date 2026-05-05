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
	vars, err := applyUserVars(runpfile.Vars, c.StringSlice(`var`))
	if err != nil {
		return err
	}
	wd, err := os.Getwd()
	if err != nil {
		ui.WriteLinef("Failed to resolve current working directory: %v", err)
	}
	vars[`runp_root`] = runpfile.Root
	vars[`runp_workdir`] = wd
	vars[`runp_file_separator`] = string(os.PathSeparator)
	runpfile.Vars = vars

	secretKey, err := resolveSecretKey(c.String(`key-env`), c.String(`key`))
	if err != nil {
		return err
	}
	runpfile.SecretKey = secretKey

	preconditions := runpfile.Preconditions
	preconditionVerifyResult := preconditions.Verify()
	if preconditionVerifyResult.Vote != core.Proceed {
		ui.WriteLinef("Preconditions failed: %s", preconditionVerifyResult.Reasons)
		return exitErrorf(3, "Preconditions failed: %s", preconditionVerifyResult.Reasons)
	}

	ui.Debugf("Starting execution with Runpfile root: %s", runpfile.Root)
	executor := core.NewExecutor(runpfile)
	if err := executor.Start(); err != nil {
		return exitErrorf(3, "Failed to execute Runpfile: %s", c.String("f"))
	}
	return nil
}

func applyUserVars(vars map[string]string, userVars []string) (map[string]string, error) {
	if len(vars) == 0 && len(userVars) > 0 {
		return nil, exitErrorf(4, "Variables provided via --var but Runpfile has no 'vars:' section: declare variable names under 'vars:' in the Runpfile before using --var")
	}
	for _, v := range userVars {
		kv := strings.SplitN(v, `=`, 2)
		if len(kv) != 2 {
			return nil, exitErrorf(4, "Invalid --var value %q: expected key=value", v)
		}
		if _, declared := vars[kv[0]]; !declared {
			return nil, exitErrorf(4, "Unknown variable %q: not declared in Runpfile", kv[0])
		}
		vars[kv[0]] = kv[1]
	}
	if len(vars) == 0 {
		vars = make(map[string]string)
	}
	return vars, nil
}

func resolveSecretKey(kev, key string) (string, error) {
	if kev != "" && key != "" {
		return "", exitErrorf(3, "Options --key and --key-env are mutually exclusive")
	}
	if kev != "" {
		ev := os.Getenv(kev)
		if ev == "" {
			return "", exitErrorf(3, "Environment variable %s is empty", kev)
		}
		return ev, nil
	}
	return key, nil
}
