package qac

import (
	"fmt"
	"io/ioutil"
	"strings"
)

// OutputAssertion represents
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
		result.addErrorf("%s: expected empty but got\n%s", a.id, out)
	}

	if a.EqualsTo != "" {
		et := strings.TrimSpace(a.EqualsTo)
		if out != et {
			result.addErrorf("%s: actual \n_%s_\nnot equal to:\n_%s_", a.id, out, et)
		}
	}
	if a.EqualsToFile != "" {
		f, err := resolvePath(a.EqualsToFile, context)
		if err != nil {
			result.addError(err)
			return result
		}
		content, err := ioutil.ReadFile(f)
		if err != nil {
			result.addError(err)
			return result
		}
		// Convert []byte to string and print to screen
		t := strings.TrimSpace(string(content))
		if out != t {
			result.addErrorf("%s: actual \n_%s_\nnot equal to:\n_%s_", a.id, out, t)
		}
	}
	if a.StartsWith != "" {
		if !strings.HasPrefix(out, a.StartsWith) {
			result.addErrorf("%s: actual output\n%s\ndoes not start with:\n%s", a.id, out, a.StartsWith)
		}
	}
	if a.EndsWith != "" {
		if !strings.HasSuffix(out, a.EndsWith) {
			result.addErrorf("%s: actual output\n%s\ndoes not end with:\n%s", a.id, out, a.EndsWith)
		}
	}
	if len(a.ContainsAll) > 0 {
		for _, t := range a.ContainsAll {
			if !strings.Contains(out, t) {
				result.addErrorf("%s: actual output\n[%s]\ndoes not contain:\n%s", a.id, out, t)
			}
		}
	}
	if len(a.ContainsAny) > 0 {
		// fail := true
		// for _, t := range a.ContainsAny {
		// 	if strings.Contains(out, t) {
		// 		fail = false
		// 		break
		// 	}
		// }
		if a.failContainsAny(out) {
			result.addErrorf("%s: actual output\n%s\ndoes not contain any of :\n%q", a.id, out, a.ContainsAny)
		}
	}

	return result
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
