package qac

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// DryRun validates the plan structure and reports which specs would run or be
// skipped, without executing any commands or touching the filesystem.
// The returned report uses InfoType entries for specs that would run and
// SkippedType entries for those that would not. Config errors produce ErrorType
// entries just as they do in Execute.
func (l *Launcher) DryRun(plan TestPlan, opts ...RunOption) *TestExecutionReport {
	cfg := applyOptions(opts)
	report := &TestExecutionReport{}
	basedir, err := os.Getwd()
	if err != nil {
		report.addEntryAsError("load", asInfraError(fmt.Errorf("getting working directory: %w", err)))
		return report
	}
	return l.dryRunPlan(plan, basedir, cfg)
}

func (l *Launcher) dryRunPlan(plan TestPlan, basedir string, cfg runConfig) *TestExecutionReport {
	report := &TestExecutionReport{}

	// Validate plan-level setup and teardown commands.
	for _, cmd := range plan.Setup {
		if err := validateCommand(cmd); err != nil {
			report.addEntryAsError("setup", asConfigError(err))
		}
	}
	for _, cmd := range plan.Teardown {
		if err := validateCommand(cmd); err != nil {
			report.addEntryAsError("teardown", asConfigError(err))
		}
	}

	order := plan.specOrder
	if len(order) == 0 {
		for key := range plan.Specs {
			order = append(order, key)
		}
	}
	numSpecs := len(order)
	for i, key := range order {
		spec, ok := plan.Specs[key]
		if !ok {
			continue
		}
		spec.id = key
		phase := specPhase(spec)
		report.openBlock(phase, i+1, numSpecs, time.Now())

		if reason := tagSkipReason(spec, cfg); reason != "" {
			report.addEntrySkipped(phase, reason)
			report.closeBlock(phase, 0)
			continue
		}
		if reason := specSkipReason(spec); reason != "" {
			report.addEntrySkipped(phase, reason)
			report.closeBlock(phase, 0)
			continue
		}

		// Validate spec-level setup and teardown.
		for _, cmd := range spec.Setup {
			if err := validateCommand(cmd); err != nil {
				report.addEntryAsError(phase, asConfigError(fmt.Errorf("setup: %w", err)))
			}
		}
		for _, cmd := range spec.Teardown {
			if err := validateCommand(cmd); err != nil {
				report.addEntryAsError(phase, asConfigError(fmt.Errorf("teardown: %w", err)))
			}
		}

		// Validate the spec command itself.
		if err := validateCommand(spec.Command); err != nil {
			report.addEntryAsError(phase, asConfigError(err))
			report.closeBlock(phase, 0)
			continue
		}

		// Validate retry_delay if set.
		if spec.RetryDelay != "" {
			if _, err := time.ParseDuration(spec.RetryDelay); err != nil {
				report.addEntryAsError(phase, asConfigError(fmt.Errorf("invalid retry_delay %q: %w", spec.RetryDelay, err)))
				report.closeBlock(phase, 0)
				continue
			}
		}

		// Spec is structurally valid: report what would be executed.
		report.addEntryInfo(phase, fmt.Sprintf("would execute: %s", commandSummary(spec.Command)))
		report.closeBlock(phase, 0)
	}
	return report
}

// ListSpecs returns the names of specs in execution order after applying the
// given RunOptions (tag filters, static skip conditions). Only specs that
// would actually run are included; already-skipped specs are omitted.
func (l *Launcher) ListSpecs(plan TestPlan, opts ...RunOption) []string {
	cfg := applyOptions(opts)
	order := plan.specOrder
	if len(order) == 0 {
		for key := range plan.Specs {
			order = append(order, key)
		}
	}
	result := make([]string, 0, len(order))
	for _, key := range order {
		spec, ok := plan.Specs[key]
		if !ok {
			continue
		}
		spec.id = key
		if tagSkipReason(spec, cfg) != "" || specSkipReason(spec) != "" {
			continue
		}
		result = append(result, key)
	}
	return result
}

// validateCommand returns an error if the command has a configuration conflict.
func validateCommand(cmd Command) error {
	if cmd.Cli != "" && cmd.Exe != "" {
		return fmt.Errorf("cli and exe are mutually exclusive")
	}
	if cmd.Stdin != "" && cmd.StdinFile != "" {
		return fmt.Errorf("stdin and stdin_file are mutually exclusive")
	}
	return nil
}

// commandSummary returns a short human-readable description of the command.
func commandSummary(cmd Command) string {
	if cmd.Cli != "" {
		return cmd.Cli
	}
	if cmd.Exe != "" {
		parts := append([]string{cmd.Exe}, cmd.Args...)
		return strings.TrimSpace(strings.Join(parts, " "))
	}
	return "(no command)"
}
