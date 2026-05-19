package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/enr/runp/lib/core"
	"github.com/urfave/cli/v2"
)

func doList(c *cli.Context) error {
	runpfile, err := loadRunpfile(resolveRunpfileArg(c), resolveChecksumArg(c))
	if err != nil {
		return err
	}

	entries := core.ListEntries(runpfile)

	switch strings.ToLower(c.String("output")) {
	case "json":
		return printListJSON(entries)
	default:
		printListTable(entries)
		return nil
	}
}

func printListTable(entries []core.UnitListEntry) {
	ui.WriteLinef("Units defined in Runpfile:")

	if len(entries) == 0 {
		ui.WriteLinef("  (none)")
		return
	}

	nameW := len("NAME")
	kindW := len("KIND")
	cmdW := len("COMMAND")
	descW := len("DESCRIPTION")
	precW := len("PRECONDITIONS")
	for _, e := range entries {
		if len(e.Name) > nameW {
			nameW = len(e.Name)
		}
		if len(e.Kind) > kindW {
			kindW = len(e.Kind)
		}
		if len(e.Command) > cmdW {
			cmdW = len(e.Command)
		}
		if len(e.Description) > descW {
			descW = len(e.Description)
		}
		if len(e.Preconditions) > precW {
			precW = len(e.Preconditions)
		}
	}

	rowFmt := fmt.Sprintf("  %%-%ds  %%-%ds  %%-%ds  %%-%ds  %%s\n", nameW, kindW, cmdW, descW)
	header := fmt.Sprintf(rowFmt, "NAME", "KIND", "COMMAND", "DESCRIPTION", "PRECONDITIONS")
	sepLen := 2 + nameW + 2 + kindW + 2 + cmdW + 2 + descW + 2 + precW
	sep := strings.Repeat("-", sepLen)

	fmt.Fprint(os.Stdout, header)
	fmt.Fprintln(os.Stdout, sep)
	for _, e := range entries {
		fmt.Fprintf(os.Stdout, rowFmt, e.Name, e.Kind, e.Command, e.Description, e.Preconditions)
	}
}

func printListJSON(entries []core.UnitListEntry) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(entries)
}
