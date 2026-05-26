package qac

import "fmt"

// tagSkipReason returns a non-empty string if spec should be skipped based on
// the tag filter in cfg, or "" if it should run.
// skipTags take precedence: a spec is excluded if it carries any skip tag,
// even if it also matches a withTags filter.
func tagSkipReason(spec Spec, cfg runConfig) string {
	for _, st := range cfg.skipTags {
		for _, t := range spec.Tags {
			if t == st {
				return fmt.Sprintf("tag %q excluded", st)
			}
		}
	}
	if len(cfg.withTags) > 0 {
		for _, wt := range cfg.withTags {
			for _, t := range spec.Tags {
				if t == wt {
					return ""
				}
			}
		}
		return fmt.Sprintf("tags %v not matched", cfg.withTags)
	}
	return ""
}
