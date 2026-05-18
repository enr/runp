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
	Name:  "up",
	Usage: "up [--var K=V] [--key KEY] [--key-env KEYENV] [--file RUNPFILE]",
	Description: `Start all processes defined in the Runpfile. This is the default command: invoking runp without a subcommand is equivalent to runp up.`,
	UsageText: `runp up
   runp up --file ./infra/Runpfile
   runp up --var DB_HOST=localhost --var DB_PORT=5432
   runp up --key-env RUNP_SECRET_KEY
   runp up --key-env RUNP_SECRET_KEY --var ENV=production --file ./prod/Runpfile`,
	Action: doUp,
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
	UsageText: `runp encrypt --key-env RUNP_SECRET_KEY mysecretvalue
   runp encrypt --key myplaintextkey mysecretvalue
   runp encrypt mysecretvalue   # generates and prints a random key`,
	Action: doEncrypt,
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
	UsageText: `runp status
   runp ps
   runp status --file ./infra/Runpfile`,
	Action: doStatus,
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Value: configFileBaseName, Usage: `Path to Runpfile`},
	},
}

var commandValidate = cli.Command{
	Name:        "validate",
	Usage:       "validate",
	Description: `Validate the Runpfile without starting any process. Exits 0 if valid, 1 if validation errors are found, 2 if the file cannot be loaded.`,
	UsageText: `runp validate
   runp validate --file ./infra/Runpfile`,
	Action: doValidate,
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
	UsageText: `runp config show
   runp config get container_runner
   runp config set container_runner podman`,
	Subcommands: []*cli.Command{
		{
			Name:        "get",
			Usage:       "get <key>",
			Description: `Print the current value of a settings key (returns the default if not explicitly set).`,
			UsageText:   `runp config get container_runner`,
			Action:      doConfigGet,
		},
		{
			Name:        "set",
			Usage:       "set <key> <value>",
			Description: `Write a settings key to ~/.runp/settings.yaml. Creates the file if it does not exist.`,
			UsageText:   `runp config set container_runner podman`,
			Action:      doConfigSet,
		},
		{
			Name:        "show",
			Usage:       "show",
			Description: `Print the full contents of ~/.runp/settings.yaml together with its path.`,
			UsageText:   `runp config show`,
			Action:      doConfigShow,
		},
	},
}

var commandReload = cli.Command{
	Name:        "reload",
	Usage:       "reload <unit-name>",
	Description: `Stop and restart a single unit by name. Run this in a separate shell while runp up is running to reload a single service without stopping the full stack. SSH tunnel units are not supported.`,
	UsageText: `runp reload web
   runp reload api --file ./infra/Runpfile`,
	Action: doReload,
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Value: configFileBaseName, Usage: `Path to Runpfile`},
	},
}

var commandList = cli.Command{
	Name:        "list",
	Aliases:     []string{"ls"},
	Usage:       "list",
	Description: `List all units defined in the Runpfile`,
	UsageText: `runp list
   runp ls
   runp list --output json
   runp list --output json | jq '.[].name'
   runp list --file ./infra/Runpfile`,
	Action: doList,
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Value: configFileBaseName, Usage: `Path to Runpfile`},
		&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Value: "table", Usage: `Output format: table (default) or json`},
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
	ui.WriteLinef("Loaded: %s", runpfilePath)
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

// resolveLogLevel derives a LogLevel from the --log-level flag and the
// deprecated --debug / --quiet aliases. --log-level takes precedence when
// set to a non-default value; aliases are checked afterwards.
func resolveLogLevel(c *cli.Context) (core.LogLevel, error) {
	ls := c.String("log-level")
	if ls != "" && ls != "info" {
		return core.ParseLogLevel(ls)
	}
	if c.Bool("debug") {
		return core.LogLevelDebug, nil
	}
	if c.Bool("quiet") {
		return core.LogLevelWarn, nil
	}
	return core.LogLevelInfo, nil
}
