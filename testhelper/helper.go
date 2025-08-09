package testhelper

import (
	"regexp"
	"strings"
	"testing"
)

var (
	whiteSpaces = regexp.MustCompile(`(\s+)`)
	leadingTabs = regexp.MustCompile(`^(\t+)`)
)

func replaceTab(match string) string {
	numTabs := strings.Count(match, "\t")  // マッチした文字列中のタブの数をカウント
	return strings.Repeat("    ", numTabs) // その数だけ4スペースを繰り返す
}

func TrimIndent(t *testing.T, src string) string {
	t.Helper()

	lines := strings.Split(src, "\n")

	var indent string
	if len(lines) > 1 {
		indent = whiteSpaces.FindString(lines[1])
	}

	for i, line := range lines {
		line = strings.TrimPrefix(line, indent)
		lines[i] = leadingTabs.ReplaceAllStringFunc(line, replaceTab)
	}

	return strings.Join(lines[1:], "\n")
}
