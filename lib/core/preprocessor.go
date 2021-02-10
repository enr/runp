package core

import (
	"fmt"
	"regexp"
)

var (
	varsRegexp = regexp.MustCompile(`{{[[:space:]]{0,}vars[[:space:]]+([a-z_]+)[[:space:]]{0,}?}}`)
)

func newCliPreprocessor(vars map[string]string) *cliPreprocessor {
	tp := &cliPreprocessor{
		vars:             vars,
		notFoundTemplate: "{!notfound key '%s'!}",
	}
	return tp
}

type cliPreprocessor struct {
	re               *regexp.Regexp
	vars             map[string]string
	notFoundTemplate string
}

func (p *cliPreprocessor) processArgs(args []string) []string {
	vsf := make([]string, 0)
	for _, v := range args {
		vsf = append(vsf, p.process(v))
	}
	return vsf
}

func (p *cliPreprocessor) process(s string) string {
	p.re = varsRegexp
	return p.re.ReplaceAllStringFunc(s, func(m string) string {
		parts := p.re.FindStringSubmatch(m)
		return p.sub(parts[1])
	})
}

func (p *cliPreprocessor) sub(s string) string {
	if val, ok := p.vars[s]; ok {
		return val
	}
	// should be strategy: leave alone, substitute, throw error...
	return fmt.Sprintf(p.notFoundTemplate, s)
}
