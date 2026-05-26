package qac

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// verify checks the command output against the assertion's expected values.
func (a *OutputAssertion) verify(context planContext) AssertionResult {
	result := AssertionResult{
		description: fmt.Sprintf(`output %s for %s`, a.id, context.commandResult.execution),
	}
	out := context.commandResult.stdout
	if a.id == `stderr` {
		out = context.commandResult.stderr
	}
	out = strings.TrimSpace(out)
	shouldBeEmpty := a.IsEmpty != nil && *a.IsEmpty
	if shouldBeEmpty && out != "" {
		result.addErrorf("%s: expected empty but got\n%s", a.id, snippet(out))
	}
	if !a.verifyEquals(out, context, &result) {
		return result
	}
	a.verifyPrefixSuffix(out, &result)
	a.verifyContains(out, &result)
	a.verifyLineCounts(out, &result)
	a.verifyRegex(out, &result)
	return result
}

// verifyEquals checks EqualsTo and EqualsToFile assertions. Returns false if a fatal infra error occurred.
func (a *OutputAssertion) verifyEquals(out string, context planContext, result *AssertionResult) bool {
	if a.EqualsTo != "" {
		et := strings.TrimSpace(a.EqualsTo)
		if out != et {
			result.addErrorf("%s: actual\n%s\nnot equal to:\n%s", a.id, snippet(out), snippet(et))
		}
	}
	if a.EqualsToFile != "" {
		f, err := resolvePath(a.EqualsToFile, context)
		if err != nil {
			result.addInfraError(fmt.Errorf("resolving equals_to_file path %q: %w", a.EqualsToFile, err))
			return false
		}
		content, err := os.ReadFile(f)
		if err != nil {
			result.addInfraError(fmt.Errorf("reading equals_to_file %q: %w", f, err))
			return false
		}
		t := strings.TrimSpace(string(content))
		if out != t {
			result.addErrorf("%s: actual\n%s\nnot equal to:\n%s", a.id, snippet(out), snippet(t))
		}
	}
	return true
}

func (a *OutputAssertion) verifyPrefixSuffix(out string, result *AssertionResult) {
	if a.StartsWith != "" {
		if !strings.HasPrefix(out, a.StartsWith) {
			result.addErrorf("%s: output does not start with: %q\n%s", a.id, a.StartsWith, contextHint(out, a.StartsWith))
		}
	}
	if a.EndsWith != "" {
		if !strings.HasSuffix(out, a.EndsWith) {
			result.addErrorf("%s: output does not end with: %q\n%s", a.id, a.EndsWith, contextHintTail(out))
		}
	}
}

func (a *OutputAssertion) verifyContains(out string, result *AssertionResult) {
	if len(a.ContainsAll) > 0 {
		for _, t := range a.ContainsAll {
			if !strings.Contains(out, t) {
				result.addErrorf("%s: output does not contain: %q\n%s", a.id, t, contextHint(out, t))
			}
		}
	}
	if len(a.ContainsAny) > 0 {
		if a.failContainsAny(out) {
			result.addErrorf("%s: output does not contain any of: %q\n%s", a.id, a.ContainsAny, contextHint(out, ""))
		}
	}
	if len(a.ContainsNone) > 0 {
		for _, t := range a.ContainsNone {
			if strings.Contains(out, t) {
				result.addErrorf("%s: output should not contain: %q\n%s", a.id, t, contextHintAround(out, t))
			}
		}
	}
	if a.ContainsLine != "" {
		found := false
		for _, line := range outputLines(out) {
			if line == a.ContainsLine {
				found = true
				break
			}
		}
		if !found {
			result.addErrorf("%s: no line equals:\n%s", a.id, a.ContainsLine)
		}
	}
}

func (a *OutputAssertion) verifyLineCounts(out string, result *AssertionResult) {
	if a.LineCount != nil {
		got := len(outputLines(out))
		if got != *a.LineCount {
			result.addErrorf("%s: line count %d != expected %d", a.id, got, *a.LineCount)
		}
	}
	if a.LineCountGte != nil {
		got := len(outputLines(out))
		if got < *a.LineCountGte {
			result.addErrorf("%s: line count %d < required %d", a.id, got, *a.LineCountGte)
		}
	}
}

func (a *OutputAssertion) verifyRegex(out string, result *AssertionResult) {
	if a.Matches != "" {
		re, err := regexp.Compile(a.Matches)
		if err != nil {
			result.addConfigError(fmt.Errorf("%s: invalid regex in matches %q: %w", a.id, a.Matches, err))
		} else if !re.MatchString(out) {
			result.addErrorf("%s: actual output\n%s\ndoes not match:\n%s", a.id, snippet(out), a.Matches)
		}
	}
	if a.NotMatches != "" {
		re, err := regexp.Compile(a.NotMatches)
		if err != nil {
			result.addConfigError(fmt.Errorf("%s: invalid regex in not_matches %q: %w", a.id, a.NotMatches, err))
		} else if re.MatchString(out) {
			result.addErrorf("%s: actual output\n%s\nshould not match:\n%s", a.id, snippet(out), a.NotMatches)
		}
	}
}

func (a *OutputAssertion) failContainsAny(out string) bool {
	fail := true
	for _, t := range a.ContainsAny {
		if strings.Contains(out, t) {
			fail = false
			break
		}
	}
	return fail
}

// outputLines splits trimmed output into lines, returning nil for empty output.
func outputLines(out string) []string {
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}
