package core

import (
	"fmt"
	"sort"
	"strings"
)

// UnitListEntry is a row in the list output.
type UnitListEntry struct {
	Name          string `json:"name"`
	Kind          string `json:"kind"`
	Command       string `json:"command"`
	Description   string `json:"description"`
	Preconditions string `json:"preconditions"`
}

// ListEntries returns a sorted slice of UnitListEntry for every unit in rf.
func ListEntries(rf *Runpfile) []UnitListEntry {
	entries := make([]UnitListEntry, 0, len(rf.Units))
	for _, unit := range rf.Units {
		entries = append(entries, UnitListEntry{
			Name:          unit.Name,
			Kind:          unitKind(unit),
			Command:       unitCommandSummary(unit),
			Description:   unit.Description,
			Preconditions: preconditionSummary(unit.Preconditions),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})
	return entries
}

// preconditionSummary produces a compact, single-line description of a unit's
// precondition set. Returns "-" when no preconditions are defined.
func preconditionSummary(p Preconditions) string {
	var parts []string

	if p.Os.IsSet() {
		parts = append(parts, fmt.Sprintf("os:%s", strings.Join(p.Os.Inclusions, ",")))
	}
	if p.Runp.IsSet() {
		op := operatorSymbol(p.Runp.Operator)
		parts = append(parts, fmt.Sprintf("runp:%s%s", op, p.Runp.Version))
	}
	if p.EnvVars.IsSet() {
		names := make([]string, 0, len(p.EnvVars.EnvVars))
		for _, ev := range p.EnvVars.EnvVars {
			names = append(names, ev.Name)
		}
		parts = append(parts, fmt.Sprintf("env:%s", strings.Join(names, ",")))
	}
	if p.Hosts.IsSet() {
		hosts := make([]string, 0, len(p.Hosts.Contains))
		for h := range p.Hosts.Contains {
			hosts = append(hosts, h)
		}
		sort.Strings(hosts)
		parts = append(parts, fmt.Sprintf("hosts:%s", strings.Join(hosts, ",")))
	}

	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, " ")
}

func operatorSymbol(op VersionComparationOperator) string {
	switch op {
	case LessThan:
		return "<"
	case LessThanOrEqual:
		return "<="
	case Equal:
		return "="
	case GreaterThanOrEqual:
		return ">="
	case GreaterThan:
		return ">"
	}
	return ""
}
