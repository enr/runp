package main

import (
	"fmt"
	"os"
	"os/signal"

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
	<-ch
	appContext.SetShuttingDown()
	runningProcesses := appContext.GetRunningProcesses()
	ui.Debug("Initiating graceful shutdown sequence...")
	if len(runningProcesses) == 0 {
		ui.Debug("No active processes to terminate.")
		os.Exit(0)
	}
	ui.Debugf("Active processes detected:")
	for _, process := range runningProcesses {
		ui.Debugf("- %s", process.ID())
	}

	for _, process := range runningProcesses {
		ui.WriteLinef("Terminating process %s", process.ID())
		cmd, err := process.StopCommand()
		if err != nil {
			ui.WriteLinef("Failed to load stop command for process %s: %v\n", process.ID(), err)
			continue
		}
		// Start() calls Stop() which implements graceful shutdown internally
		if err := cmd.Start(); err != nil {
			ui.WriteLinef("Failed to execute stop command for process %s: %v\n", process.ID(), err)
			continue
		}
		// Wait for the stop command to complete (Stop() already handles timeout internally)
		err = cmd.Wait()
		if err != nil {
			ui.WriteLinef("Process %s stopped with error: %v\n", process.ID(), err)
		} else {
			ui.Debugf("Process %s stopped successfully\n", process.ID())
		}
	}

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
	signal.Notify(ch, os.Interrupt)
	go listenForShutdown(ch)

	app := cli.NewApp()
	app.Name = "runp"
	app.Version = appVersion
	app.Usage = "Run processes"
	app.Flags = []cli.Flag{
		&cli.BoolFlag{Name: "debug", Aliases: []string{"d"}, Usage: "operates in debug mode: lot of output"},
		&cli.BoolFlag{Name: "quiet", Aliases: []string{"q"}, Usage: "operates in quiet mode"},
		&cli.BoolFlag{Name: "no-color", Aliases: []string{"C"}, Usage: "no colors in output"},
	}
	app.EnableBashCompletion = true

	app.Before = func(c *cli.Context) error {
		debug := c.Bool("debug")
		_, noColorEnv := os.LookupEnv("NO_COLOR")
		avoidColor := noColorEnv || c.Bool("no-color")
		colorize := !avoidColor
		ui = core.CreateMainLogger(" ", 6, "%s> ", debug, colorize)
		processLoggerConfiguration := core.LoggerConfig{
			Debug: debug,
			Color: colorize,
		}
		core.ConfigureUI(ui, processLoggerConfiguration)
		return nil
	}

	app.Commands = commands

	app.Run(os.Args)
}
