package qac

import (
	"os"
	"strings"
)

// interpolate replaces ${name} placeholders in src.
//
// Resolution order:
//  1. Plan-level vars (from the "vars:" section).
//  2. System environment variables, reachable as ${env.NAME}.
//
// Placeholders that match nothing are left unchanged.
func interpolate(src []byte, vars map[string]string) []byte {
	s := string(src)
	for k, v := range vars {
		s = strings.ReplaceAll(s, "${"+k+"}", v)
	}
	for _, pair := range os.Environ() {
		idx := strings.IndexByte(pair, '=')
		if idx < 0 {
			continue
		}
		name, val := pair[:idx], pair[idx+1:]
		placeholder := "${env." + name + "}"
		if strings.Contains(s, placeholder) {
			s = strings.ReplaceAll(s, placeholder, val)
		}
	}
	return []byte(s)
}
