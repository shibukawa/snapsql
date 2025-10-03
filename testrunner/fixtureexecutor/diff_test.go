package fixtureexecutor

import (
	"os"
	"strings"
	"testing"

	"github.com/fatih/color"
)

func TestMain(m *testing.M) {
	color.NoColor = true

	os.Exit(m.Run())
}

func TestFormatDiffUnifiedYAML(t *testing.T) {
	// 色付け無効化はTestMainで一括設定
	diff := &DiffError{
		Table:       "lists",
		PrimaryKeys: []string{"id"},
		RowDiffs: []RowDiff{
			{
				Key: map[string]any{"id": 10},
				Diffs: []ColumnDiff{{
					Column:   "name",
					Expected: "Todo",
					Actual:   "Todo!",
					Reason:   "value mismatch",
				}},
			},
		},
	}

	diffText := FormatDiffUnifiedYAML(diff)
	if diffText == "" {
		t.Fatalf("expected diff output, got empty string")
	}

	checks := []string{
		"Table: lists",
		"- Expected",
		"+ Actual",
		"id: 10 [mismatch]",
		"+ name: Todo",
		"- name: Todo!",
	}
	for _, want := range checks {
		if !strings.Contains(diffText, want) {
			t.Fatalf("expected diff output to contain %q\n%s", want, diffText)
		}
	}

	for _, notWant := range []string{"detail:", "row_index"} {
		if strings.Contains(diffText, notWant) {
			t.Fatalf("diff output should not contain %q\n%s", notWant, diffText)
		}
	}
}
