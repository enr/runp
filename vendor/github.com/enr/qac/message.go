package qac

import (
	"fmt"
	"os"
	"strings"
)

const snippetMaxLen = 400

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
