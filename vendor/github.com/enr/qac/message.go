package qac

import (
	"fmt"
	"os"
	"strings"
)

const (
	snippetMaxLen = 400
	contextRadius = 3
)

func snippet(s string) string {
	if os.Getenv("QAC_VERBOSE") != "" || len(s) <= snippetMaxLen {
		return s
	}
	nLines := strings.Count(s, "\n") + 1
	return fmt.Sprintf("%s\n[...%d chars, %d lines; set QAC_VERBOSE=1 for full output]",
		s[:snippetMaxLen], len(s), nLines)
}

func snippetTail(s string) string {
	if os.Getenv("QAC_VERBOSE") != "" || len(s) <= snippetMaxLen {
		return s
	}
	nLines := strings.Count(s, "\n") + 1
	tail := s[len(s)-snippetMaxLen:]
	return fmt.Sprintf("[...%d chars, %d lines; last %d bytes]\n%s",
		len(s), nLines, snippetMaxLen, tail)
}

// contextHint returns a numbered line window around the location where target
// is "closest" to appearing in text. Replaces full-content dumps in "not found"
// error messages.
func contextHint(text, target string) string {
	lines := splitLines(text)
	if len(lines) == 0 {
		return "  (empty)"
	}
	return contextWindow(lines, bestMatchLine(lines, target))
}

// contextHintAround returns a numbered line window around where target actually
// appears in text. Used when the content IS found but should not be.
func contextHintAround(text, target string) string {
	lines := splitLines(text)
	if len(lines) == 0 {
		return "  (empty)"
	}
	needle := firstLine(target)
	center := 0
	for i, line := range lines {
		if strings.Contains(line, needle) {
			center = i
			break
		}
	}
	return contextWindow(lines, center)
}

// contextHintTail returns a numbered line window anchored at the end of text.
func contextHintTail(text string) string {
	lines := splitLines(text)
	if len(lines) == 0 {
		return "  (empty)"
	}
	return contextWindow(lines, len(lines)-1)
}

func contextWindow(lines []string, center int) string {
	n := len(lines)
	from := center - contextRadius
	if from < 0 {
		from = 0
	}
	to := center + contextRadius
	if to >= n {
		to = n - 1
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "  context (lines %d-%d of %d):", from+1, to+1, n)
	for i := from; i <= to; i++ {
		line := lines[i]
		if len(line) > 120 {
			line = line[:117] + "..."
		}
		fmt.Fprintf(&sb, "\n    %d: %s", i+1, line)
	}
	return sb.String()
}

// bestMatchLine returns the index of the line in lines where target has the
// longest prefix overlap — i.e., the line most likely near where target should appear.
func bestMatchLine(lines []string, target string) int {
	if target == "" {
		return 0
	}
	needle := firstLine(target)
	best, bestScore := 0, 0
	for i, line := range lines {
		if s := prefixOverlap(needle, line); s > bestScore {
			bestScore = s
			best = i
		}
	}
	return best
}

// prefixOverlap returns the length of the longest prefix of needle that
// appears as a substring of haystack.
func prefixOverlap(needle, haystack string) int {
	for l := len(needle); l > 0; l-- {
		if strings.Contains(haystack, needle[:l]) {
			return l
		}
	}
	return 0
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func splitLines(s string) []string {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
