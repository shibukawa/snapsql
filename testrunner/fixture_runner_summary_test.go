package testrunner

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/shibukawa/snapsql/testrunner/fixtureexecutor"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	prevStdout := os.Stdout
	prevColorOutput := color.Output

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	os.Stdout = w
	color.Output = w

	done := make(chan string)

	go func() {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, r); err != nil {
			done <- ""
			return
		}

		done <- buf.String()
	}()

	fn()
	w.Close()

	os.Stdout = prevStdout
	color.Output = prevColorOutput

	return <-done
}

func TestFixtureTestRunnerPrintSummaryLabelsFailures(t *testing.T) {
	t.Parallel()

	color.NoColor = true

	t.Cleanup(func() {
		color.NoColor = false
	})

	runner := &FixtureTestRunner{}
	runner.SetVerbose(true)

	diffErr := &fixtureexecutor.DiffError{
		Table:       "lists",
		PrimaryKeys: []string{"id"},
		RowDiffs: []fixtureexecutor.RowDiff{
			{
				Key: map[string]any{"id": 10},
				Diffs: []fixtureexecutor.ColumnDiff{{
					Column:   "name",
					Expected: "Todo",
					Actual:   "Todo!",
				}},
			},
		},
	}

	summary := &FixtureTestSummary{
		TotalTests:    3,
		PassedTests:   0,
		FailedTests:   3,
		TotalDuration: 1500 * time.Millisecond,
		Results: []FixtureTestResult{
			{
				TestName:    "Assertion failure case",
				Success:     false,
				Error:       fixtureexecutor.NewFixtureFailure(fixtureexecutor.FailureKindAssertion, diffErr),
				FailureKind: fixtureexecutor.FailureKindAssertion,
				SourceFile:  "tests/case.md",
				SourceLine:  42,
				ExecutedSQL: []fixtureexecutor.SQLTrace{
					{
						Label:      "main query",
						Statement:  "SELECT * FROM comments WHERE card_id = 401 ORDER BY created_at",
						Parameters: map[string]any{"card_id": 401},
						Args:       []any{401},
						QueryType:  fixtureexecutor.SelectQuery,
						Rows: []map[string]any{{
							"body":       "Looks good",
							"card_id":    401,
							"created_at": "2025-09-14T11:00:00Z",
							"id":         600,
						}},
						TotalRows: 1,
					},
				},
			},
			{
				TestName:    "Definition failure case",
				Success:     false,
				Error:       fixtureexecutor.NewFixtureFailure(fixtureexecutor.FailureKindDefinition, io.EOF),
				FailureKind: fixtureexecutor.FailureKindDefinition,
				SourceFile:  "tests/case.md",
				SourceLine:  108,
				ExecutedSQL: []fixtureexecutor.SQLTrace{
					{
						Label:      "main query",
						Statement:  "UPDATE cards SET title = ? WHERE id = ? 0",
						Parameters: map[string]any{"id": 99},
						Args:       []any{"New Title", 99},
						QueryType:  fixtureexecutor.UpdateQuery,
					},
				},
			},
			{
				TestName:    "Unknown failure case",
				Success:     false,
				Error:       io.EOF,
				FailureKind: fixtureexecutor.FailureKindUnknown,
				SourceFile:  "tests/other.md",
				SourceLine:  7,
			},
		},
		AssertionFailures:  1,
		DefinitionFailures: 1,
		UnknownFailures:    1,
	}

	output := captureStdout(t, func() {
		runner.PrintSummary(summary)
	})

	if !strings.Contains(output, "Assertions Failed: 1") {
		t.Fatalf("expected assertion failure count in output, got: %s", output)
	}

	if !strings.Contains(output, "Definition Failures: 1") {
		t.Fatalf("expected definition failure count in output, got: %s", output)
	}

	if !strings.Contains(output, "Unknown Failures: 1") {
		t.Fatalf("expected unknown failure count in output, got: %s", output)
	}

	if !strings.Contains(output, "⚠️ [Failure] Assertion failure case (tests/case.md:42)") {
		t.Fatalf("expected failure label in output, got: %s", output)
	}

	if !strings.Contains(output, "❌ [Error] Definition failure case (tests/case.md:108)") {
		t.Fatalf("expected error label in output, got: %s", output)
	}

	if !strings.Contains(output, "❔ [Unknown] Unknown failure case (tests/other.md:7)") {
		t.Fatalf("expected unknown label in output, got: %s", output)
	}

	if !strings.Contains(output, "Table: lists") {
		t.Fatalf("expected table header in output, got: %s", output)
	}

	if !strings.Contains(output, "- Expected") || !strings.Contains(output, "+ Actual") {
		t.Fatalf("expected legend lines in output, got: %s", output)
	}

	if !strings.Contains(output, "id: 10 [mismatch]") || !strings.Contains(output, "+ name: Todo") || !strings.Contains(output, "- name: Todo!") {
		t.Fatalf("expected diff body in output, got: %s", output)
	}

	if !strings.Contains(output, "SQL Trace:") {
		t.Fatalf("expected SQL trace header in output, got: %s", output)
	}

	if !strings.Contains(output, "Statement: SELECT * FROM comments WHERE card_id = 401 ORDER BY created_at") {
		t.Fatalf("expected SQL statement in output, got: %s", output)
	}

	if !strings.Contains(output, "Params:") {
		t.Fatalf("expected params section in output, got: %s", output)
	}

	if !strings.Contains(output, "Args:") {
		t.Fatalf("expected args section in output, got: %s", output)
	}

	if !strings.Contains(output, "[1]: 401") {
		t.Fatalf("expected positional args in output, got: %s", output)
	}

	if !strings.Contains(output, "Statement: UPDATE cards SET title = ? WHERE id = ? 0") {
		t.Fatalf("expected definition failure SQL statement in output, got: %s", output)
	}

	if strings.Contains(output, "detail:") {
		t.Fatalf("diff output should not include detail field: %s", output)
	}

	if strings.Contains(output, "row_index") {
		t.Fatalf("diff output should not include row_index: %s", output)
	}
}
