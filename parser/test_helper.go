package parser

import (
	"regexp"
	"strings"
	"testing"
)

var whiteSpaces = regexp.MustCompile(`(\s+)`)

func TrimIndent(t *testing.T, src string) string {
	t.Helper()
	lines := strings.Split(src, "\n")
	var indent string
	if len(lines) > 1 {
		indent = whiteSpaces.FindString(lines[1])
	}
	for i, line := range lines {
		lines[i] = strings.TrimPrefix(line, indent)
	}
	return strings.Join(lines[1:], "\n")
}
