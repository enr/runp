package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/enr/runp/lib/core"
	"github.com/urfave/cli/v2"
)

func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// fetchRunpfilePath resolves f to a local file path, downloading it when f is
// an HTTP/HTTPS URL. For URLs the returned cleanup removes the temp file; for
// local paths cleanup is a no-op. displayPath is always f (the original value).
func fetchRunpfilePath(f, checksum string) (localPath, displayPath string, cleanup func(), err error) {
	cleanup = func() {}
	displayPath = f
	if isURL(f) {
		var tmp string
		tmp, err = core.FetchRunpfile(f, checksum)
		if err != nil {
			return
		}
		localPath = tmp
		cleanup = func() { os.Remove(tmp) }
		return
	}
	localPath, err = core.ResolveRunpfilePath(f)
	return
}

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
	&commandCompletion,
}

var commandUp = cli.Command{
	Name:        "up",
	Usage:       "up [--var K=V] [--key KEY] [--key-env KEYENV] [--file RUNPFILE]",
	Description: `Start all processes defined in the Runpfile. This is the default command: invoking runp without a subcommand is equivalent to runp up.`,
	UsageText: `runp up
   runp up --file ./infra/Runpfile
   runp up --var DB_HOST=localhost --var DB_PORT=5432
   runp up --key-env RUNP_SECRET_KEY
   runp up --key-env RUNP_SECRET_KEY --var ENV=production --file ./prod/Runpfile`,
	Action: doUp,
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Value: configFileBaseName, Usage: `Path to Runpfile or HTTP/HTTPS URL (overrides RUNP_FILE env var)`},
		&cli.StringFlag{Name: "checksum", Usage: `Expected SHA-256 checksum of the Runpfile in "sha256:<hex>" format; verifies --file when it is a URL (required for http:// URLs)`},
		&cli.BoolFlag{Name: "dry-run", Aliases: []string{"n"}, Usage: "Print what would be executed without starting any process"},
		&cli.StringSliceFlag{Name: "var", Aliases: []string{"V"}, Usage: `Runtime variables in format "key=value"`},
		&cli.StringFlag{Name: "key", Aliases: []string{"k"}, Usage: `Decryption key (WARNING: visible in 'ps aux' and shell history — prefer --key-env)`},
		&cli.StringFlag{Name: "key-env", Usage: `Name of the environment variable containing the decryption key (recommended)`},
	},
}

var commandEncrypt = cli.Command{
	Name:  "encrypt",
	Usage: "encrypt [--key-env KEYENV | --key KEY] [SECRET]",
	Description: `Encrypt a secret value for use in Runpfile. The secret can be passed as a
positional argument or piped via stdin.

SECURITY NOTE: avoid --key with a literal value — the key will be visible in
'ps aux' output and recorded in your shell history. Use --key-env to pass the
key through an environment variable instead:

  export RUNP_SECRET_KEY=my-encryption-key
  runp encrypt --key-env RUNP_SECRET_KEY mysecretvalue`,
	UsageText: `runp encrypt --key-env RUNP_SECRET_KEY mysecretvalue
   echo mysecretvalue | runp encrypt --key-env RUNP_SECRET_KEY
   runp encrypt mysecretvalue   # generates and prints a random key
   runp encrypt --key myplaintextkey mysecretvalue  # WARNING: key visible in ps/history`,
	Action: doEncrypt,
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "key", Aliases: []string{"k"}, Usage: `Encryption key (WARNING: visible in 'ps aux' and shell history — prefer --key-env)`},
		&cli.StringFlag{Name: "key-env", Usage: `Name of the environment variable containing the encryption key (recommended)`},
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
		&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Value: configFileBaseName, Usage: `Path to Runpfile or HTTP/HTTPS URL (overrides RUNP_FILE env var)`},
		&cli.StringFlag{Name: "checksum", Usage: `Expected SHA-256 checksum in "sha256:<hex>" format (required for http:// URLs)`},
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
		&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Value: configFileBaseName, Usage: `Path to Runpfile or HTTP/HTTPS URL (overrides RUNP_FILE env var)`},
		&cli.StringFlag{Name: "checksum", Usage: `Expected SHA-256 checksum in "sha256:<hex>" format (required for http:// URLs)`},
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
   runp reload --file ./infra/Runpfile api
   runp reload --var DB_HOST=localhost --key-env RUNP_SECRET_KEY api

   Flags must precede the unit name.`,
	Action: doReload,
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Value: configFileBaseName, Usage: `Path to Runpfile or HTTP/HTTPS URL (overrides RUNP_FILE env var)`},
		&cli.StringFlag{Name: "checksum", Usage: `Expected SHA-256 checksum in "sha256:<hex>" format (required for http:// URLs)`},
		&cli.StringSliceFlag{Name: "var", Aliases: []string{"V"}, Usage: `Runtime variables in format "key=value" (pass the same values used with runp up)`},
		&cli.StringFlag{Name: "key", Aliases: []string{"k"}, Usage: `Decryption key (WARNING: visible in 'ps aux' and shell history — prefer --key-env)`},
		&cli.StringFlag{Name: "key-env", Usage: `Name of the environment variable containing the decryption key (recommended)`},
	},
	BashComplete: completionUnitNames,
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
		&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Value: configFileBaseName, Usage: `Path to Runpfile or HTTP/HTTPS URL (overrides RUNP_FILE env var)`},
		&cli.StringFlag{Name: "checksum", Usage: `Expected SHA-256 checksum in "sha256:<hex>" format (required for http:// URLs)`},
		&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Value: "table", Usage: `Output format: table (default) or json`},
	},
}

func exitError(exitCode int, message string) error {
	return cli.NewExitError(message, exitCode)
}

// when error it returns an `exitError`
func loadRunpfile(f, checksum string) (*core.Runpfile, error) {
	localPath, displayPath, cleanup, err := fetchRunpfilePath(f, checksum)
	defer cleanup()
	if err != nil {
		if isURL(f) {
			return &core.Runpfile{}, exitErrorf(exitCodeLoad, "Failed to fetch Runpfile: %s", err)
		}
		return &core.Runpfile{}, runpfileNotFoundError(localPath)
	}
	ui.WriteLinef("Loaded: %s", displayPath)
	runpfile, err := core.LoadRunpfileFromPath(localPath)
	if err != nil {
		return &core.Runpfile{}, exitErrorf(exitCodeLoad,
			"Cannot parse Runpfile at %s:\n  %s", displayPath, err.Error())
	}
	valid, errs := core.IsRunpfileValid(runpfile)
	if !valid {
		var b strings.Builder
		fmt.Fprintf(&b, "Invalid Runpfile %s:\n", displayPath)
		for _, e := range errs {
			fmt.Fprintf(&b, "  - %s\n", e.Error())
		}
		b.WriteString("  → Run `runp validate` for the full report")
		return &core.Runpfile{}, exitErrorf(exitCodeLoad, "%s", b.String())
	}
	return runpfile, nil
}

func exitErrorf(exitCode int, template string, args ...interface{}) error {
	return cli.NewExitError(fmt.Sprintf(template, args...), exitCode)
}

// runpfileNotFoundError prints the "not found" message to stdout and returns a
// silent exit-code-2 error. Output goes to stdout (not stderr) so that callers
// can capture it via combined output or stdout-only redirection.
func runpfileNotFoundError(resolvedPath string) error {
	fmt.Fprintf(os.Stdout,
		"Runpfile not found at %s\n  → Run `runp init` to create one, or use --file to specify a path\n",
		resolvedPath)
	return cli.Exit("", exitCodeLoad)
}

// resolveRunpfileArg returns the Runpfile path to use for a command,
// implementing the precedence: --file flag > RUNP_FILE env var > default.
//
// It walks the full context lineage (subcommand → global) so that
// "runp -f path" (flag at global level) and "runp up -f path" (flag at
// subcommand level) both work when 'up' is the default command.
//
// A non-default value that differs from the compiled-in default is treated
// as explicitly set — this handles test helpers that inject a path as the
// flag's default value without going through flag.Parse.
func resolveRunpfileArg(c *cli.Context) string {
	// Walk innermost → outermost context to find the first explicitly-set value.
	for _, ctx := range c.Lineage() {
		if ctx.IsSet("file") {
			return ctx.String("file")
		}
		if ctx.IsSet("f") {
			return ctx.String("f")
		}
	}
	// Fallback: non-default value in the current context covers test helpers
	// that set a specific path as the flag default without flag.Parse.
	if v := c.String("file"); v != "" && v != configFileBaseName {
		return v
	}
	if v := c.String("f"); v != "" && v != configFileBaseName {
		return v
	}
	if env := os.Getenv("RUNP_FILE"); env != "" {
		return env
	}
	return configFileBaseName
}

// resolveChecksumArg returns the --checksum value from the first context in the
// lineage that has it set, or empty string if not provided.
func resolveChecksumArg(c *cli.Context) string {
	for _, ctx := range c.Lineage() {
		if ctx.IsSet("checksum") {
			return ctx.String("checksum")
		}
	}
	return ""
}

// boolFromLineage returns true if any of the named flags was explicitly set
// in any context in the lineage (subcommand → global). This is needed when
// the same bool flag is defined at both global and subcommand level so that
// "runp --flag" and "runp up --flag" both work correctly.
func boolFromLineage(c *cli.Context, names ...string) bool {
	for _, ctx := range c.Lineage() {
		for _, name := range names {
			if ctx.IsSet(name) {
				return true
			}
		}
	}
	return false
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
