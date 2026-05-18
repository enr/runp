package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/enr/runp/lib/core"
	"github.com/urfave/cli/v2"
)

func doStatus(c *cli.Context) error {
	runpfile, err := loadRunpfile(resolveRunpfileArg(c))
	if err != nil {
		return err
	}

	envSettings := core.LoadEnvironmentSettings()
	pidDir, _ := core.PIDDirForRoot(runpfile.Root)

	statuses := make([]core.UnitStatus, 0, len(runpfile.Units))
	for _, unit := range runpfile.Units {
		statuses = append(statuses, core.ProbeStatus(unit, pidDir, envSettings))
	}
	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].Name < statuses[j].Name
	})

	printStatusTable(statuses)
	return nil
}

func printStatusTable(statuses []core.UnitStatus) {
	if len(statuses) == 0 {
		fmt.Fprintln(os.Stdout, "No units defined in Runpfile.")
		return
	}

	nameW := len("NAME")
	kindW := len("KIND")
	stateW := len("STATE")
	for _, s := range statuses {
		if len(s.Name) > nameW {
			nameW = len(s.Name)
		}
		if len(s.Kind) > kindW {
			kindW = len(s.Kind)
		}
		if len(string(s.State)) > stateW {
			stateW = len(string(s.State))
		}
	}

	rowFmt := fmt.Sprintf("%%-%ds  %%-%ds  %%-%ds  %%s\n", nameW, kindW, stateW)
	header := fmt.Sprintf(rowFmt, "NAME", "KIND", "STATE", "DETAIL")
	sepLen := nameW + kindW + stateW + 3*2 // columns + separators
	sep := make([]byte, sepLen)
	for i := range sep {
		sep[i] = '-'
	}

	fmt.Fprint(os.Stdout, header)
	fmt.Fprintln(os.Stdout, string(sep))
	for _, s := range statuses {
		fmt.Fprintf(os.Stdout, rowFmt, s.Name, s.Kind, string(s.State), s.Detail)
	}
}
