package fixtureexecutor

import (
	"errors"
	"fmt"
	"maps"
	"sort"
	"strings"

	"github.com/fatih/color"
)

// FailureKind represents the classification of a fixture failure.
type FailureKind int

const (
	// FailureKindUnknown represents failures that could not be classified.
	FailureKindUnknown FailureKind = iota
	// FailureKindAssertion represents assertion/validation mismatches.
	FailureKindAssertion
	// FailureKindDefinition represents fixture definition/setup failures.
	FailureKindDefinition
)

var (
	tableHeaderFmt     = color.New(color.FgBlue, color.Bold).SprintfFunc()
	legendExpectedFmt  = color.New(color.FgGreen).SprintFunc()
	legendActualFmt    = color.New(color.FgRed).SprintFunc()
	primaryKeyNameFmt  = color.New(color.FgBlue, color.Bold, color.Underline).SprintfFunc()
	primaryKeyValueFmt = color.New(color.FgBlue, color.Underline).SprintfFunc()
	rowLabelFmt        = color.New(color.FgBlue, color.Bold).SprintfFunc()
	expectPrefixFmt    = color.New(color.BgGreen, color.FgBlack).SprintFunc()
	actualPrefixFmt    = color.New(color.BgRed, color.FgBlack).SprintFunc()
	expectFieldFmt     = color.New(color.FgGreen).SprintfFunc()
	actualFieldFmt     = color.New(color.FgRed).SprintfFunc()
	expectValueFmt     = color.New(color.BgGreen, color.FgBlack).SprintfFunc()
	actualValueFmt     = color.New(color.BgRed, color.FgBlack).SprintfFunc()
	expectSeparatorFmt = color.New(color.FgGreen).SprintFunc()
	actualSeparatorFmt = color.New(color.FgRed).SprintFunc()
)

// Sentinel errors for wrapping (err113 compliant)
var (
	ErrUnknownFixtureFailure = errors.New("unknown fixture failure")
	ErrFixtureFailureMessage = errors.New("fixture failure")
)

// FixtureError is an error wrapper that retains the failure classification and optional context.
type FixtureError struct {
	kind    FailureKind
	err     error
	context map[string]string
}

// NOTE: Historical alias "FixtureFailure" was removed to satisfy errname linter.
// Keep helper functions to avoid breaking call sites while returning *FixtureError.

// Error implements the error interface.
func (f *FixtureError) Error() string {
	if f == nil || f.err == nil {
		return "fixture failure"
	}

	return f.err.Error()
}

// Unwrap returns the underlying error.
func (f *FixtureError) Unwrap() error {
	if f == nil {
		return nil
	}

	return f.err
}

// Kind returns the FailureKind classification.
func (f *FixtureError) Kind() FailureKind {
	if f == nil {
		return FailureKindUnknown
	}

	return f.kind
}

// Context returns a copy of the contextual metadata attached to the failure.
func (f *FixtureError) Context() map[string]string {
	if f == nil || len(f.context) == 0 {
		return nil
	}

	out := make(map[string]string, len(f.context))
	maps.Copy(out, f.context)

	return out
}

type DiffError struct {
	Table            string
	PrimaryKeys      []string
	RowCountMismatch bool
	ExpectedRows     int
	ActualRows       int
	RowDiffs         []RowDiff
}

type RowDiff struct {
	Key       map[string]any
	Diffs     []ColumnDiff
	RowStatus string
}

type ColumnDiff struct {
	Column   string
	Expected any
	Actual   any
	Reason   string
}

func (d *DiffError) Error() string {
	if d == nil {
		return "diff error"
	}

	return "diff detected"
}

// NewFixtureFailure creates a new FixtureFailure with the provided kind.
func NewFixtureFailure(kind FailureKind, err error) error {
	return newFixtureFailure(kind, err, nil)
}

// AsFixtureFailure attempts to extract a FixtureError from the error chain.
func AsFixtureFailure(err error) (*FixtureError, bool) {
	var ff *FixtureError
	if errors.As(err, &ff) {
		return ff, true
	}

	return nil, false
}

// AsDiffError attempts to extract a DiffError from the error chain.
func AsDiffError(err error) (*DiffError, bool) {
	var de *DiffError
	if errors.As(err, &de) {
		return de, true
	}

	return nil, false
}

// ClassifyFailure inspects an error and returns its FailureKind.
func ClassifyFailure(err error) FailureKind {
	if err == nil {
		return FailureKindUnknown
	}

	if ff, ok := AsFixtureFailure(err); ok {
		return ff.Kind()
	}

	msg := strings.ToLower(err.Error())

	assertionPrefixes := []string{
		"simple validation failed",
		"verify query validation failed",
		"table state validation failed",
		"validation failed",
	}

	definitionPrefixes := []string{
		"failed to execute fixtures",
		"failed to execute fixture",
		"failed to execute verify query",
		"failed to execute query",
		"failed to execute dml query",
		"failed to execute main sql",
		"failed to load fixture external file",
		"failed to load expected results",
		"failed to query table",
		"failed to get column names",
		"failed to scan row",
		"failed to count rows",
		"failed to clear table",
		"failed to insert row",
		"failed to prepare insert statement",
		"failed to execute delete",
		"failed to unmarshal external rows",
	}

	for _, prefix := range assertionPrefixes {
		if strings.HasPrefix(msg, prefix) {
			return FailureKindAssertion
		}
	}

	for _, prefix := range definitionPrefixes {
		if strings.HasPrefix(msg, prefix) {
			return FailureKindDefinition
		}
	}

	return FailureKindUnknown
}

func newFixtureFailure(kind FailureKind, err error, ctx map[string]string) error {
	if err == nil {
		err = ErrUnknownFixtureFailure
	}

	return &FixtureError{kind: kind, err: err, context: copyContext(ctx)}
}

func wrapFailure(kind FailureKind, err error, ctx map[string]string, format string, args ...any) error {
	message := fmt.Sprintf(format, args...)
	merged := copyContext(ctx)

	if ff, ok := AsFixtureFailure(err); ok {
		if existing := ff.Context(); len(existing) > 0 {
			if merged == nil {
				merged = make(map[string]string, len(existing))
			}

			for k, v := range existing {
				if _, present := merged[k]; !present {
					merged[k] = v
				}
			}
		}
	}

	if err == nil {
		return newFixtureFailure(kind, fmt.Errorf("%w: %s", ErrFixtureFailureMessage, message), merged)
	}

	return newFixtureFailure(kind, fmt.Errorf("%s: %w", message, err), merged)
}

func wrapAssertionFailure(err error, format string, args ...any) error {
	return wrapFailure(FailureKindAssertion, err, nil, format, args...)
}

func wrapDefinitionFailure(err error, format string, args ...any) error {
	return wrapFailure(FailureKindDefinition, err, nil, format, args...)
}

func wrapDefinitionFailureWithContext(ctx map[string]string, err error, format string, args ...any) error {
	return wrapFailure(FailureKindDefinition, err, ctx, format, args...)
}

func copyContext(ctx map[string]string) map[string]string {
	if len(ctx) == 0 {
		return nil
	}

	out := make(map[string]string, len(ctx))
	maps.Copy(out, ctx)

	return out
}

// FormatDiffUnifiedYAML renders a DiffError as a compact textual report ready for CLI output.
func FormatDiffUnifiedYAML(diff *DiffError) string {
	if diff == nil {
		return ""
	}

	if !diff.RowCountMismatch && len(diff.RowDiffs) == 0 {
		return ""
	}

	var b strings.Builder

	if diff.Table != "" {
		b.WriteString(tableHeaderFmt("Table: %s\n", diff.Table))
	}

	b.WriteString(legendExpectedFmt("- Expected\n"))
	b.WriteString(legendActualFmt("+ Actual\n"))

	if diff.RowCountMismatch {
		b.WriteString(expectFieldFmt("+ rows: %d\n", diff.ExpectedRows))
		b.WriteString(actualFieldFmt("- rows: %d\n", diff.ActualRows))
	}

	rowDiffs := make([]RowDiff, len(diff.RowDiffs))
	copy(rowDiffs, diff.RowDiffs)
	sort.Slice(rowDiffs, func(i, j int) bool {
		return formatKey(rowDiffs[i].Key) < formatKey(rowDiffs[j].Key)
	})

	for idx, row := range rowDiffs {
		writePrimaryKeyLine(&b, row, idx)
		writeRowDifferences(&b, row)
		b.WriteString("\n")
	}

	return strings.TrimRight(b.String(), "\n")
}

func writePrimaryKeyLine(b *strings.Builder, row RowDiff, index int) {
	keys := make([]string, 0, len(row.Key))
	for k := range row.Key {
		if k == "row_index" {
			continue
		}

		keys = append(keys, k)
	}

	if len(keys) == 0 {
		b.WriteString(rowLabelFmt("row #%d%s\n", index+1, rowStatusSuffix(row)))
		return
	}

	sort.Strings(keys)

	for i, key := range keys {
		if i > 0 {
			b.WriteString(primaryKeyValueFmt(", "))
		}

		b.WriteString(primaryKeyNameFmt("%s", key))
		val := formatDiffScalar(formatValueForDiff(row.Key[key]))
		b.WriteString(primaryKeyValueFmt(": %s", val))
	}

	b.WriteString(rowStatusSuffix(row))
	b.WriteString("\n")
}

func writeRowDifferences(b *strings.Builder, row RowDiff) {
	columns := make([]ColumnDiff, len(row.Diffs))
	copy(columns, row.Diffs)
	sort.Slice(columns, func(i, j int) bool {
		return columns[i].Column < columns[j].Column
	})

	expectedFields := buildFieldList(columns, true)
	actualFields := buildFieldList(columns, false)

	includeExpected := row.RowStatus != "unexpected" && len(expectedFields) > 0
	includeActual := row.RowStatus != "missing" && len(actualFields) > 0

	if row.RowStatus == "unexpected" {
		expectedFields = []string{expectFieldFmt("(no expected row)")}
		includeExpected = true
	}

	if row.RowStatus == "missing" {
		actualFields = []string{actualFieldFmt("(no actual row)")}
		includeActual = true
	}

	if includeExpected {
		writeFieldLine(b, expectPrefixFmt, "+", expectSeparatorFmt, expectedFields)
	}

	if includeActual {
		writeFieldLine(b, actualPrefixFmt, "-", actualSeparatorFmt, actualFields)
	}
}

func rowStatusSuffix(row RowDiff) string {
	status := ""

	switch row.RowStatus {
	case "missing":
		status = " [missing]"
	case "unexpected":
		status = " [unexpected]"
	}

	if status == "" && len(row.Diffs) > 0 {
		status = " [mismatch]"
	}

	return status
}

func buildFieldList(columns []ColumnDiff, expected bool) []string {
	fields := make([]string, 0, len(columns))
	for _, col := range columns {
		label := col.Column
		if label == "__row__" {
			label = "row"
		}

		var value any
		if expected {
			value = col.Expected
		} else {
			value = col.Actual
		}

		value = formatValueForDiff(value)

		valStr := formatDiffScalar(value)
		if expected {
			fields = append(fields, expectFieldFmt("%s: %s", label, expectValueFmt("%s", valStr)))
		} else {
			fields = append(fields, actualFieldFmt("%s: %s", label, actualValueFmt("%s", valStr)))
		}
	}

	return fields
}

func writeFieldLine(b *strings.Builder, prefix func(...any) string, sign string, sep func(...any) string, fields []string) {
	if len(fields) == 0 {
		return
	}

	b.WriteString(prefix(sign))
	b.WriteString(" ")

	for i, field := range fields {
		if i > 0 {
			b.WriteString(sep(", "))
		}

		b.WriteString(field)
	}

	b.WriteString("\n")
}

func formatDiffScalar(v any) string {
	if v == nil {
		return "<nil>"
	}

	return fmt.Sprintf("%v", v)
}

func formatKey(key map[string]any) string {
	if len(key) == 0 {
		return ""
	}

	keys := make([]string, 0, len(key))
	for k := range key {
		if k == "row_index" {
			continue
		}

		keys = append(keys, k)
	}

	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, key[k]))
	}

	return strings.Join(parts, ", ")
}
