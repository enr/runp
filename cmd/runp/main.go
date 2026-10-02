package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/enr/runp/lib/core"
	"github.com/urfave/cli/v2"
)

var (
	ui              core.Logger
	versionTemplate = `%s
Revision: %s
Build date: %s
`
	appVersion = fmt.Sprintf(versionTemplate, core.Version, core.GitCommit, core.BuildTime)
	appContext = core.GetApplicationContext()
)

func listenForShutdown(ch <-chan os.Signal) {
	select {
	case <-ch:
	case <-appContext.ShutdownChan():
	}
	ui.Debug("Initiating graceful shutdown sequence")
	appContext.StopRunningProcesses()

	// Universal ANSI sequences (compatible with Windows 10+ and Linux)
	// Block 1: Reset colors and attributes
	resetColors := "\033[0m"
	// Block 2: Show cursor
	showCursor := "\033[?25h"
	// Block 3: Exit alternate screen buffer
	exitAltBuffer := "\033[?1049l"
	// Block 4: Position cursor at the bottom
	moveFarDown := "\033[1000B"
	// Block 5: Move cursor far to the left
	moveFarLeft := "\033[1000D"
	// Block 6: Clear entire screen
	// clearScreen := "\033[2J"
	// Block 7: Move cursor to top-left position (1;1)
	// homeCursor := "\033[1;1H"
	// Block 8: Final color reset and add empty lines for screen cleanup
	resetColorsFinal := "\033[0m\n\n\n"

	// Compose complete reset sequence
	resetSequence := resetColors +
		showCursor +
		exitAltBuffer +
		moveFarDown +
		moveFarLeft +
		// clearScreen +
		// homeCursor +
		resetColorsFinal

	fmt.Print(resetSequence)
	os.Exit(0)
}

func main() {
	// manage stop signals
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go listenForShutdown(ch)

	app := cli.NewApp()
	app.Name = "runp"
	app.Version = appVersion
	app.Usage = "Run processes defined in Runpfile (default command: up)"
	app.Flags = []cli.Flag{
		&cli.StringFlag{Name: "log-level", Aliases: []string{"l"}, Value: "info", Usage: "Log verbosity: trace, debug, info (default), warn, error"},
		&cli.BoolFlag{Name: "debug", Aliases: []string{"d"}, Usage: "Deprecated: use --log-level debug"},
		&cli.BoolFlag{Name: "quiet", Aliases: []string{"q"}, Usage: "Deprecated: use --log-level warn"},
		&cli.BoolFlag{Name: "no-color", Aliases: []string{"C"}, Usage: "Disable colored output"},
		&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Value: configFileBaseName, Usage: `Path to Runpfile — used when running the default 'up' command without an explicit subcommand`},
		&cli.BoolFlag{Name: "dry-run", Aliases: []string{"n"}, Usage: "Print what would be executed without starting any process"},
	}
	app.EnableBashCompletion = true

	app.Before = func(c *cli.Context) error {
		level, err := resolveLogLevel(c)
		if err != nil {
			return cli.NewExitError(err.Error(), 3)
		}
		_, noColorEnv := os.LookupEnv("NO_COLOR")
		avoidColor := noColorEnv || c.Bool("no-color")
		colorize := !avoidColor
		ui = core.CreateMainLoggerWithLevel(" ", 6, "%s> ", level, colorize)
		core.ConfigureUI(ui, core.LoggerConfig{Level: level, Color: colorize})
		return nil
	}

	app.Commands = commands
	app.DefaultCommand = "up"

	app.Run(os.Args)
}
