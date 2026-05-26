package qac

// OutputMatcher is a function that configures an OutputAssertion.
// Use the constructor functions (Contains, Equals, Empty, etc.) to create matchers.
type OutputMatcher func(*OutputAssertion)

// --- Command helpers ---

// Cmd creates a Command that runs the given executable directly (no shell).
func Cmd(exe string, args ...string) Command {
	return Command{Exe: exe, Args: args}
}

// ShellCmd creates a Command that runs the given shell command line.
func ShellCmd(cli string) Command {
	return Command{Cli: cli}
}

// --- OutputMatcher constructors ---

// Contains requires the output to contain all of the given substrings.
func Contains(substrings ...string) OutputMatcher {
	return func(a *OutputAssertion) {
		a.ContainsAll = append(a.ContainsAll, substrings...)
	}
}

// ContainsAnyOf requires the output to contain at least one of the given substrings.
func ContainsAnyOf(substrings ...string) OutputMatcher {
	return func(a *OutputAssertion) {
		a.ContainsAny = append(a.ContainsAny, substrings...)
	}
}

// NotContains requires the output to contain none of the given substrings.
func NotContains(substrings ...string) OutputMatcher {
	return func(a *OutputAssertion) {
		a.ContainsNone = append(a.ContainsNone, substrings...)
	}
}

// Equals requires the output to equal the given string exactly (after trimming).
func Equals(s string) OutputMatcher {
	return func(a *OutputAssertion) {
		a.EqualsTo = s
	}
}

// Empty requires the output to be empty.
func Empty() OutputMatcher {
	t := true
	return func(a *OutputAssertion) {
		a.IsEmpty = &t
	}
}

// StartsWith requires the output to start with the given prefix (after trimming).
func StartsWith(prefix string) OutputMatcher {
	return func(a *OutputAssertion) {
		a.StartsWith = prefix
	}
}

// EndsWith requires the output to end with the given suffix (after trimming).
func EndsWith(suffix string) OutputMatcher {
	return func(a *OutputAssertion) {
		a.EndsWith = suffix
	}
}

// Matches requires the entire trimmed output to match the given regular expression.
func Matches(re string) OutputMatcher {
	return func(a *OutputAssertion) {
		a.Matches = re
	}
}

// NotMatches requires the entire trimmed output NOT to match the given regular expression.
func NotMatches(re string) OutputMatcher {
	return func(a *OutputAssertion) {
		a.NotMatches = re
	}
}

// HasLineCount requires the output to have exactly n non-empty lines.
func HasLineCount(n int) OutputMatcher {
	return func(a *OutputAssertion) {
		a.LineCount = &n
	}
}

// HasLineCountAtLeast requires the output to have at least n non-empty lines.
func HasLineCountAtLeast(n int) OutputMatcher {
	return func(a *OutputAssertion) {
		a.LineCountGte = &n
	}
}

// ContainsExactLine requires at least one line to equal the given string exactly (after trimming).
func ContainsExactLine(line string) OutputMatcher {
	return func(a *OutputAssertion) {
		a.ContainsLine = line
	}
}

// --- SpecBuilder ---

// SpecBuilder is a fluent builder for Spec.
type SpecBuilder struct {
	spec Spec
}

// NewSpec creates a new SpecBuilder.
func NewSpec() *SpecBuilder {
	return &SpecBuilder{}
}

// Description sets the human-readable description for the spec.
func (b *SpecBuilder) Description(d string) *SpecBuilder {
	b.spec.Description = d
	return b
}

// Tags sets the tags for the spec.
func (b *SpecBuilder) Tags(tags ...string) *SpecBuilder {
	b.spec.Tags = append(b.spec.Tags, tags...)
	return b
}

// Skip marks this spec so it is always skipped.
func (b *SpecBuilder) Skip() *SpecBuilder {
	b.spec.Skip = true
	return b
}

// Retries sets the number of additional attempts after the first failure.
func (b *SpecBuilder) Retries(n int) *SpecBuilder {
	b.spec.Retries = n
	return b
}

// RetryDelay sets the delay between retries (e.g. "1s", "500ms").
func (b *SpecBuilder) RetryDelay(d string) *SpecBuilder {
	b.spec.RetryDelay = d
	return b
}

// Command sets the command under test.
func (b *SpecBuilder) Command(cmd Command) *SpecBuilder {
	b.spec.Command = cmd
	return b
}

// Setup appends setup commands that run before this spec.
func (b *SpecBuilder) Setup(cmds ...Command) *SpecBuilder {
	b.spec.Setup = append(b.spec.Setup, cmds...)
	return b
}

// Teardown appends teardown commands that run after this spec.
func (b *SpecBuilder) Teardown(cmds ...Command) *SpecBuilder {
	b.spec.Teardown = append(b.spec.Teardown, cmds...)
	return b
}

// ExpectStatus asserts the command exits with the given status code.
func (b *SpecBuilder) ExpectStatus(code int) *SpecBuilder {
	b.spec.Expectations.StatusAssertion.EqualsTo = &code
	return b
}

// ExpectStatusGT asserts the exit status is greater than n.
func (b *SpecBuilder) ExpectStatusGT(n int) *SpecBuilder {
	b.spec.Expectations.StatusAssertion.GreaterThan = &n
	return b
}

// ExpectStatusLT asserts the exit status is less than n.
func (b *SpecBuilder) ExpectStatusLT(n int) *SpecBuilder {
	b.spec.Expectations.StatusAssertion.LessThan = &n
	return b
}

// ExpectStdout adds one or more matchers on the command's standard output.
func (b *SpecBuilder) ExpectStdout(matchers ...OutputMatcher) *SpecBuilder {
	for _, m := range matchers {
		m(&b.spec.Expectations.OutputAssertions.Stdout)
	}
	return b
}

// ExpectStderr adds one or more matchers on the command's standard error.
func (b *SpecBuilder) ExpectStderr(matchers ...OutputMatcher) *SpecBuilder {
	for _, m := range matchers {
		m(&b.spec.Expectations.OutputAssertions.Stderr)
	}
	return b
}

// Build returns the constructed Spec value.
func (b *SpecBuilder) Build() Spec {
	return b.spec
}

// --- PlanBuilder ---

// PlanBuilder is a fluent builder for TestPlan.
type PlanBuilder struct {
	plan TestPlan
}

// NewPlan creates a new PlanBuilder.
func NewPlan() *PlanBuilder {
	return &PlanBuilder{plan: TestPlan{
		Specs: make(map[string]Spec),
	}}
}

// Var sets a plan-level variable.
func (b *PlanBuilder) Var(key, value string) *PlanBuilder {
	if b.plan.Vars == nil {
		b.plan.Vars = make(map[string]string)
	}
	b.plan.Vars[key] = value
	return b
}

// Setup appends plan-level setup commands.
func (b *PlanBuilder) Setup(cmds ...Command) *PlanBuilder {
	b.plan.Setup = append(b.plan.Setup, cmds...)
	return b
}

// Teardown appends plan-level teardown commands.
func (b *PlanBuilder) Teardown(cmds ...Command) *PlanBuilder {
	b.plan.Teardown = append(b.plan.Teardown, cmds...)
	return b
}

// Spec adds a spec to the plan under the given name.
// The spec argument may be a *SpecBuilder (Build() is called automatically)
// or a Spec value directly.
func (b *PlanBuilder) Spec(name string, spec interface{}) *PlanBuilder {
	var s Spec
	switch v := spec.(type) {
	case *SpecBuilder:
		s = v.Build()
	case Spec:
		s = v
	}
	b.plan.Specs[name] = s
	b.plan.specOrder = append(b.plan.specOrder, name)
	return b
}

// Build returns the constructed TestPlan value.
func (b *PlanBuilder) Build() TestPlan {
	return b.plan
}
