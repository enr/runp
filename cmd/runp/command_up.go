package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/enr/runp/lib/core"
)

func doUp(c *cli.Context) error {
	if len(os.Args) == 1 {
		return cli.ShowAppHelp(c)
	}
	runpfile, err := prepareRunpfile(c)
	if err != nil {
		return err
	}

	executor := core.NewExecutor(runpfile)

	if boolFromLineage(c, "dry-run", "n") {
		return doDryRun(executor)
	}

	ui.Debugf("Starting execution with Runpfile root: %s", runpfile.Root)
	if err := executor.Start(); err != nil {
		return exitErrorf(exitCodeExec, "Failed to execute Runpfile: %s", resolveRunpfileArg(c))
	}
	return nil
}

// prepareRunpfile loads the Runpfile and applies everything needed before
// starting units: user and implicit vars, variable expansion, the secret key
// and root preconditions. "up" and "reload" share it so that a reloaded unit
// runs exactly as it would under "up".
func prepareRunpfile(c *cli.Context) (*core.Runpfile, error) {
	runpfile, err := loadRunpfile(resolveRunpfileArg(c), resolveChecksumArg(c))
	if err != nil {
		return nil, err
	}
	vars, err := applyUserVars(runpfile.Vars, c.StringSlice(`var`))
	if err != nil {
		return nil, err
	}
	wd, err := os.Getwd()
	if err != nil {
		ui.WriteLinef("Failed to resolve current working directory: %v", err)
	}
	vars[`runp_root`] = runpfile.Root
	vars[`runp_workdir`] = wd
	vars[`runp_file_separator`] = string(os.PathSeparator)
	vars, err = core.ExpandVars(vars)
	if err != nil {
		return nil, exitErrorf(exitCodeVar, "Variable expansion failed: %v", err)
	}
	runpfile.Vars = vars

	secretKey, err := resolveSecretKey(c.String(`key-env`), c.String(`key`))
	if err != nil {
		return nil, err
	}
	runpfile.SecretKey = secretKey

	preconditions := runpfile.Preconditions
	preconditionVerifyResult := preconditions.Verify()
	if preconditionVerifyResult.Vote != core.Proceed {
		return nil, exitErrorf(exitCodeExec, "Preconditions not met: %s\n  → Check that the required OS, environment variables, and runp version are satisfied", preconditionVerifyResult.Reasons)
	}
	return runpfile, nil
}

func doDryRun(executor *core.RunpfileExecutor) error {
	previews := executor.DryRunPreviews()
	if len(previews) == 0 {
		fmt.Fprintln(os.Stdout, "No units defined in Runpfile.")
		return nil
	}

	nameW := len("NAME")
	kindW := len("KIND")
	for _, p := range previews {
		if len(p.Name) > nameW {
			nameW = len(p.Name)
		}
		if len(p.Kind) > kindW {
			kindW = len(p.Kind)
		}
	}

	rowFmt := fmt.Sprintf("%%-%ds  %%-%ds  %%s\n", nameW, kindW)
	header := fmt.Sprintf(rowFmt, "NAME", "KIND", "COMMAND")
	sepLen := nameW + kindW + 2 + len("COMMAND")
	sep := strings.Repeat("-", sepLen)

	fmt.Fprintln(os.Stdout, "[dry-run] units that would be started:")
	fmt.Fprint(os.Stdout, header)
	fmt.Fprintln(os.Stdout, sep)
	for _, p := range previews {
		cmd := p.Command
		if p.Skipped {
			cmd = fmt.Sprintf("(skipped: %s)", p.SkipReason)
		}
		fmt.Fprintf(os.Stdout, rowFmt, p.Name, p.Kind, cmd)
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
