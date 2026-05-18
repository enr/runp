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
	&commandStatus,
	&commandValidate,
	&commandReload,
	&commandConfig,
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
var commandStatus = cli.Command{
	Name:        "status",
	Aliases:     []string{"ps"},
	Usage:       "status",
	Description: `Show the runtime status of all units defined in the Runpfile`,
	Action:      doStatus,
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Value: configFileBaseName, Usage: `Path to Runpfile`},
	},
}

var commandValidate = cli.Command{
	Name:        "validate",
	Usage:       "validate",
	Description: `Validate the Runpfile without starting any process. Exits 0 if valid, 1 if validation errors are found, 2 if the file cannot be loaded.`,
	Action:      doValidate,
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Value: configFileBaseName, Usage: `Path to Runpfile`},
	},
}

var commandConfig = cli.Command{
	Name:  "config",
	Usage: "config <subcommand>",
	Description: `Manage runp user settings stored in ~/.runp/settings.yaml.

Supported keys:
` + "  container_runner    Container runtime executable used for container units (default: \"docker\")" + `

Subcommands:
  get <key>          Print the current value of a setting key
  set <key> <value>  Write a setting key to the settings file
  show               Print the full settings file and its path`,
	Subcommands: []*cli.Command{
		{
			Name:        "get",
			Usage:       "get <key>",
			Description: `Print the current value of a settings key (returns the default if not explicitly set).`,
			Action:      doConfigGet,
		},
		{
			Name:        "set",
			Usage:       "set <key> <value>",
			Description: `Write a settings key to ~/.runp/settings.yaml. Creates the file if it does not exist.`,
			Action:      doConfigSet,
		},
		{
			Name:        "show",
			Usage:       "show",
			Description: `Print the full contents of ~/.runp/settings.yaml together with its path.`,
			Action:      doConfigShow,
		},
	},
}

var commandReload = cli.Command{
	Name:        "reload",
	Usage:       "reload <unit-name>",
	Description: `Stop and restart a single unit by name. Run this in a separate shell while runp up is running to reload a single service without stopping the full stack. SSH tunnel units are not supported.`,
	Action:      doReload,
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Value: configFileBaseName, Usage: `Path to Runpfile`},
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
