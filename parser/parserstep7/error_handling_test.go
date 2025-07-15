package parserstep7

import (
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestErrorHandling(t *testing.T) {
	reporter := NewErrorReporter()

	// Test adding errors
	pos := Position{Line: 10, Column: 5, File: "test.sql"}
	reporter.AddError(ErrorTypeUnresolvedReference, "Field not found", pos)

	assert.True(t, reporter.HasErrors())
	assert.False(t, reporter.HasWarnings())
	assert.Equal(t, 1, len(reporter.GetErrors()))
}

func TestErrorReporter(t *testing.T) {
	reporter := NewErrorReporter()

	pos1 := Position{Line: 1, Column: 1, File: "test.sql"}
	pos2 := Position{Line: 2, Column: 1, File: "test.sql"}

	// Add multiple errors
	reporter.AddError(ErrorTypeCircularDependency, "Circular dependency", pos1)
	reporter.AddWarning(ErrorTypeScopeViolation, "Scope issue", pos2)

	assert.True(t, reporter.HasErrors())
	assert.True(t, reporter.HasWarnings())
	assert.Equal(t, 1, len(reporter.GetErrors()))
	assert.Equal(t, 1, len(reporter.GetWarnings()))

	// Test filtering by type
	circularErrors := reporter.GetErrorsByType(ErrorTypeCircularDependency)
	assert.Equal(t, 1, len(circularErrors))
	assert.Equal(t, ErrorTypeCircularDependency, circularErrors[0].Type)
}

func TestErrorReporterMaxErrors(t *testing.T) {
	reporter := NewErrorReporter()
	reporter.maxErrors = 2 // Set low limit for testing

	pos := Position{Line: 1, Column: 1, File: "test.sql"}

	// Add more errors than the limit
	reporter.AddError(ErrorTypeUnknown, "Error 1", pos)
	reporter.AddError(ErrorTypeUnknown, "Error 2", pos)
	reporter.AddError(ErrorTypeUnknown, "Error 3", pos) // This should be ignored

	assert.Equal(t, 2, len(reporter.GetErrors()))
}

func TestParseError(t *testing.T) {
	pos := Position{Line: 10, Column: 5, File: "test.sql"}
	err := &ParseError{
		Type:        ErrorTypeUnresolvedReference,
		Message:     "Field 'unknown_field' not found",
		Position:    pos,
		Context:     "SELECT unknown_field FROM table",
		Suggestions: []string{"Check field name", "Verify table schema"},
		RelatedIDs:  []string{"field1", "field2"},
	}

	errorString := err.Error()

	assert.Contains(t, errorString, "UnresolvedReference")
	assert.Contains(t, errorString, "Field 'unknown_field' not found")
	assert.Contains(t, errorString, "test.sql:10:5")
	assert.Contains(t, errorString, "Context:")
	assert.Contains(t, errorString, "Suggestions:")
	assert.Contains(t, errorString, "Related:")
}

func TestPosition(t *testing.T) {
	// Test position with file
	pos1 := Position{Line: 10, Column: 5, File: "test.sql"}
	assert.Equal(t, "test.sql:10:5", pos1.String())

	// Test position without file
	pos2 := Position{Line: 10, Column: 5}
	assert.Equal(t, "10:5", pos2.String())
}

func TestErrorType(t *testing.T) {
	tests := []struct {
		errorType ErrorType
		expected  string
	}{
		{ErrorTypeCircularDependency, "CircularDependency"},
		{ErrorTypeUnresolvedReference, "UnresolvedReference"},
		{ErrorTypeInvalidSubquery, "InvalidSubquery"},
		{ErrorTypeScopeViolation, "ScopeViolation"},
		{ErrorTypeTypeIncompatibility, "TypeIncompatibility"},
		{ErrorTypeSyntaxError, "SyntaxError"},
		{ErrorTypeUnknown, "Unknown"},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, test.errorType.String())
	}
}

func TestErrorCollector(t *testing.T) {
	reporter := NewErrorReporter()
	collector := NewErrorCollector(reporter)

	pos := Position{Line: 1, Column: 1, File: "test.sql"}

	// Test circular dependency reporting
	path := []string{"a", "b", "c", "a"}
	collector.ReportCircularDependency(path, pos)

	errors := reporter.GetErrors()
	assert.Equal(t, 1, len(errors))
	assert.Equal(t, ErrorTypeCircularDependency, errors[0].Type)
	assert.Contains(t, errors[0].Message, "a -> b -> c -> a")
}

func TestErrorCollectorUnresolvedReference(t *testing.T) {
	reporter := NewErrorReporter()
	collector := NewErrorCollector(reporter)

	pos := Position{Line: 5, Column: 10, File: "query.sql"}
	availableFields := []string{"id", "name", "email"}

	collector.ReportUnresolvedReference("unknown_field", availableFields, pos)

	errors := reporter.GetErrors()
	assert.Equal(t, 1, len(errors))
	assert.Equal(t, ErrorTypeUnresolvedReference, errors[0].Type)
	assert.Contains(t, errors[0].Message, "unknown_field")
	assert.Contains(t, errors[0].Error(), "Available fields")
	assert.Contains(t, errors[0].Error(), "id")
	assert.Contains(t, errors[0].Error(), "name")
	assert.Contains(t, errors[0].Error(), "email")
}

func TestErrorCollectorScopeViolation(t *testing.T) {
	reporter := NewErrorReporter()
	collector := NewErrorCollector(reporter)

	pos := Position{Line: 3, Column: 15, File: "complex.sql"}

	collector.ReportScopeViolation("outer_field", "inner_scope", "outer_scope", pos)

	errors := reporter.GetErrors()
	assert.Equal(t, 1, len(errors))
	assert.Equal(t, ErrorTypeScopeViolation, errors[0].Type)
	assert.Contains(t, errors[0].Message, "outer_field")
	assert.Contains(t, errors[0].Context, "outer_scope")
	assert.Contains(t, errors[0].Context, "inner_scope")
}

func TestErrorCollectorInvalidSubquery(t *testing.T) {
	reporter := NewErrorReporter()
	collector := NewErrorCollector(reporter)

	pos := Position{Line: 8, Column: 20, File: "subquery.sql"}

	collector.ReportInvalidSubquery("sub1", "missing SELECT clause", pos)

	errors := reporter.GetErrors()
	assert.Equal(t, 1, len(errors))
	assert.Equal(t, ErrorTypeInvalidSubquery, errors[0].Type)
	assert.Contains(t, errors[0].Message, "sub1")
	assert.Contains(t, errors[0].Message, "missing SELECT clause")
	assert.Equal(t, []string{"sub1"}, errors[0].RelatedIDs)
}

func TestErrorReporterString(t *testing.T) {
	reporter := NewErrorReporter()

	pos1 := Position{Line: 1, Column: 1, File: "test.sql"}
	pos2 := Position{Line: 2, Column: 1, File: "test.sql"}

	reporter.AddError(ErrorTypeUnresolvedReference, "Error message", pos1)
	reporter.AddWarning(ErrorTypeScopeViolation, "Warning message", pos2)

	output := reporter.String()

	assert.Contains(t, output, "Errors (1):")
	assert.Contains(t, output, "Warnings (1):")
	assert.Contains(t, output, "Error message")
	assert.Contains(t, output, "Warning message")
}

func TestErrorReporterClear(t *testing.T) {
	reporter := NewErrorReporter()

	pos := Position{Line: 1, Column: 1, File: "test.sql"}
	reporter.AddError(ErrorTypeUnknown, "Error", pos)
	reporter.AddWarning(ErrorTypeUnknown, "Warning", pos)

	assert.True(t, reporter.HasErrors())
	assert.True(t, reporter.HasWarnings())

	reporter.Clear()

	assert.False(t, reporter.HasErrors())
	assert.False(t, reporter.HasWarnings())
	assert.Equal(t, 0, len(reporter.GetErrors()))
	assert.Equal(t, 0, len(reporter.GetWarnings()))
}
