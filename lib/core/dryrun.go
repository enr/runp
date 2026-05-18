package core

import (
	"fmt"
	"sort"
	"strings"
)

// UnitPreview describes what a unit would do when started.
type UnitPreview struct {
	Name       string
	Kind       string
	Command    string
	WorkDir    string
	Skipped    bool
	SkipReason string
}

// DryRunPreviews initializes all units and returns a resolved summary for each
// one without starting any process. Skipped units (unsatisfied preconditions)
// are included with Skipped=true so the caller can still show them.
func (e *RunpfileExecutor) DryRunPreviews() []UnitPreview {
	e.initializeUnits()
	skipped := e.skippedUnits()

	previews := make([]UnitPreview, 0, len(e.rf.Units))
	for _, unit := range e.rf.Units {
		process := unit.Process()
		preview := UnitPreview{
			Name:    unit.Name,
			Kind:    unitKind(unit),
			Command: unitCommandSummary(unit),
			Skipped: skipped[unit.Name],
		}
		if skipped[unit.Name] {
			preview.SkipReason = "preconditions not satisfied"
		}
		if process != nil {
			preview.WorkDir = process.Dir()
		}
		previews = append(previews, preview)
	}

	sort.Slice(previews, func(i, j int) bool {
		return previews[i].Name < previews[j].Name
	})
	return previews
}

// unitKind returns the short process-type label for a unit.
func unitKind(unit *RunpUnit) string {
	switch {
	case unit.Host != nil:
		return "host"
	case unit.Container != nil:
		return "container"
	case unit.SSHTunnel != nil:
		return "ssh_tunnel"
	}
	return "unknown"
}

// unitCommandSummary returns a human-readable, variable-resolved description
// of the command the unit would execute.
func unitCommandSummary(unit *RunpUnit) string {
	if unit.Host != nil {
		h := unit.Host
		pre := newCliPreprocessor(h.vars)
		if h.CommandLine != "" {
			return pre.process(h.CommandLine)
		}
		if h.Executable != "" {
			parts := append([]string{h.Executable}, h.Args...)
			return strings.Join(parts, " ")
		}
		return ""
	}
	if unit.Container != nil {
		c := unit.Container
		pre := newCliPreprocessor(c.vars)
		img := pre.process(c.Image)
		if len(c.Ports) > 0 {
			return fmt.Sprintf("%s  ports: %s", img, strings.Join(c.Ports, ", "))
		}
		return img
	}
	if unit.SSHTunnel != nil {
		t := unit.SSHTunnel
		return fmt.Sprintf("%s -> %s -> %s",
			t.Local.String(), t.Jump.String(), t.Target.String())
	}
	return ""
}
