package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"

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
	runningProcesses := appContext.GetRunningProcesses()
	ui.Debug("Managing shutdown...")
	if len(runningProcesses) == 0 {
		ui.Debug("No running process to close.")
		os.Exit(0)
	}
	ui.Debugf("Still alive processes:")
	for _, process := range runningProcesses {
		ui.Debugf("- %s", process.ID())
	}

	for _, process := range runningProcesses {
		ui.WriteLinef("Stop running process %s", process.ID())
		cmd, err := process.StopCommand()
		if err != nil {
			ui.WriteLinef("Error loading stopper command for process %s with error: %v\n", process.ID(), err)
			continue
		}
		duration := process.StopTimeout()
		if err := cmd.Start(); err != nil {
			ui.WriteLinef("Error calling stop command for process %s: %v\n", process.ID(), err)
		}
		f := func() {
			ui.Debugf("Force process %s kill after 5 seconds", process.ID())
			cmd.Stop()
		}

		timer := time.AfterFunc(duration, f)

		defer timer.Stop()
		err = cmd.Wait()
		if err != nil {
			ui.WriteLinef("Stopped process %s with error: %v\n", process.ID(), err)
		} else {
			ui.Debugf("Stopped %s with no error\n", process.ID())
		}
	}

    // Sequenze ANSI universali (funzionano su Windows 10+ e Linux)
    // Blocco 1: Reset colori e attributi
    resetColors := "\033[0m"
    // Blocco 2: Mostra il cursore
    showCursor := "\033[?25h"
    // Blocco 3: Esci dall'alternate screen buffer
    exitAltBuffer := "\033[?1049l"
    // Blocco 4: Posiziona il cursore molto in basso
    moveFarDown := "\033[1000B"
    // Blocco 5: Sposta molto a sinistra
    moveFarLeft := "\033[1000D"
    // Blocco 6: Pulisci tutto lo schermo
    // clearScreen := "\033[2J"
    // Blocco 7: Vai in alto a sinistra (1;1)
    // homeCursor := "\033[1;1H"
    // Blocco 8: Reset finale colori e aggiungi righe vuote per "pulire"
    resetColorsFinal := "\033[0m\n\n\n"

    // Componi la sequenza di reset completa
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
	signal.Notify(ch, os.Interrupt, os.Kill)
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
