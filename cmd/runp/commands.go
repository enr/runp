package main

import (
	"fmt"
	"strings"

	"github.com/enr/runp/lib/core"
	"github.com/urfave/cli/v2"
)

const (
	configFileBaseName = "Runpfile"
)

var commands = []*cli.Command{
	&commandUp,
	&commandEncrypt,
	&commandList,
}

var commandUp = cli.Command{
	Name:        "up",
	Usage:       "up [--var K=V] [--key KEY] [--key-env KEYENV] [--file RUNPFILE]",
	Description: `Start all processes defined in the Runpfile`,
	Action:      doUp,
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Value: configFileBaseName, Usage: `Path to Runpfile`},
		&cli.StringSliceFlag{Name: "var", Aliases: []string{"V"}, Usage: `Runtime variables in format "key=value"`},
		&cli.StringFlag{Name: "key", Aliases: []string{"k"}, Usage: `Encryption key used to decrypt secrets`},
		&cli.StringFlag{Name: "key-env", Usage: `Environment variable name containing the encryption key for secrets`},
	},
}
var commandEncrypt = cli.Command{
	Name:        "encrypt",
	Usage:       "encrypt [--key KEY] [--key-env KEYENV] SECRET",
	Description: `Encrypt a secret value for use in Runpfile`,
	Action:      doEncrypt,
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "key", Aliases: []string{"k"}, Usage: `Encryption key used to encrypt the secret`},
		&cli.StringFlag{Name: "key-env", Usage: `Environment variable name containing the encryption key`},
	},
}
var commandList = cli.Command{
	Name:        "list",
	Aliases:     []string{"ls"},
	Usage:       "list",
	Description: `List all units defined in the Runpfile`,
	Action:      doList,
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Value: configFileBaseName, Usage: `Path to Runpfile`},
	},
}

func exitError(exitCode int, message string) error {
	ui.WriteLinef("Error occurred")
	return cli.NewExitError(message, exitCode)
}

// when error it returns an `exitError`
func loadRunpfile(f string) (*core.Runpfile, error) {
	runpfilePath, err := core.ResolveRunpfilePath(f)
	if err != nil {
		return &core.Runpfile{}, exitErrorf(2, "Runpfile %s not found", runpfilePath)
	}
	ui.Debugf("Using Runpfile %s", runpfilePath)
	runpfile, err := core.LoadRunpfileFromPath(runpfilePath)
	if err != nil {
		return &core.Runpfile{}, exitErrorf(2, "Failed to load Runpfile %s: %s", runpfilePath, err.Error())
	}
	valid, errs := core.IsRunpfileValid(runpfile)
	if !valid {
		var b strings.Builder
		b.WriteString("Invalid Runpfile ")
		b.WriteString(runpfilePath)
		b.WriteString(":\n")
		for _, e := range errs {
			fmt.Fprintf(&b, "- %s\n", e.Error())
		}
		return &core.Runpfile{}, exitErrorf(2, "%s", b.String())
	}
	return runpfile, nil
}

func exitErrorf(exitCode int, template string, args ...interface{}) error {
	ui.WriteLinef("Error occurred")
	return cli.NewExitError(fmt.Sprintf(template, args...), exitCode)
}
