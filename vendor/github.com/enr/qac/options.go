package qac

// RunOption configures a single Execute or ExecuteFile call.
type RunOption func(*runConfig)

type runConfig struct {
	withTags []string
	skipTags []string
	failFast bool
}

func applyOptions(opts []RunOption) runConfig {
	cfg := runConfig{}
	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}

// WithTags restricts execution to specs that have at least one of the given tags.
// Specs with no matching tag are reported as skipped.
func WithTags(tags ...string) RunOption {
	return func(c *runConfig) {
		c.withTags = append(c.withTags, tags...)
	}
}

// SkipTags excludes specs that have at least one of the given tags.
// Excluded specs are reported as skipped.
func SkipTags(tags ...string) RunOption {
	return func(c *runConfig) {
		c.skipTags = append(c.skipTags, tags...)
	}
}

// FailFast stops execution after the first spec failure.
// Subsequent specs are not run; plan-level teardown still executes.
func FailFast() RunOption {
	return func(c *runConfig) {
		c.failFast = true
	}
}
