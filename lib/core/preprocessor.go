package core

import (
	"fmt"
	"regexp"
	"sort"
)

var (
	varsRegexp = regexp.MustCompile(`{{[[:space:]]{0,}vars[[:space:]]+([a-z_]+)[[:space:]]{0,}?}}`)
)

// ExpandVars resolves {{vars NAME}} references inside var values, enabling
// vars to reference other vars. It returns an error if a var value references
// an undeclared variable or if a circular reference is detected.
func ExpandVars(vars map[string]string) (map[string]string, error) {
	resolved := make(map[string]string, len(vars))
	visiting := make(map[string]bool)

	var resolve func(name string) (string, error)
	resolve = func(name string) (string, error) {
		if v, ok := resolved[name]; ok {
			return v, nil
		}
		if visiting[name] {
			return "", fmt.Errorf("circular variable reference detected: %q", name)
		}
		raw, ok := vars[name]
		if !ok {
			return "", fmt.Errorf("variable %q is not declared", name)
		}
		if !varsRegexp.MatchString(raw) {
			resolved[name] = raw
			return raw, nil
		}
		visiting[name] = true
		var firstErr error
		expanded := varsRegexp.ReplaceAllStringFunc(raw, func(m string) string {
			if firstErr != nil {
				return m
			}
			parts := varsRegexp.FindStringSubmatch(m)
			v, err := resolve(parts[1])
			if err != nil {
				firstErr = err
				return m
			}
			return v
		})
		visiting[name] = false
		if firstErr != nil {
			return "", firstErr
		}
		resolved[name] = expanded
		return expanded, nil
	}

	names := make([]string, 0, len(vars))
	for n := range vars {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		if _, err := resolve(name); err != nil {
			return nil, err
		}
	}
	return resolved, nil
}

func newCliPreprocessor(vars map[string]string) *cliPreprocessor {
	return &cliPreprocessor{vars: vars}
}

type cliPreprocessor struct {
	vars map[string]string
}

func (p *cliPreprocessor) processArgs(args []string) []string {
	vsf := make([]string, 0, len(args))
	for _, v := range args {
		vsf = append(vsf, p.process(v))
	}
	return vsf
}

func (p *cliPreprocessor) process(s string) string {
	return varsRegexp.ReplaceAllStringFunc(s, func(m string) string {
		parts := varsRegexp.FindStringSubmatch(m)
		if val, ok := p.vars[parts[1]]; ok {
			return val
		}
		return fmt.Sprintf("{undefined:%s}", parts[1])
	})
}
