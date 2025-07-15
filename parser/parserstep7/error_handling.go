package parserstep7

import (
	"fmt"
	"strings"
)

// ErrorType represents the type of parsing error
type ErrorType int

const (
	ErrorTypeUnknown ErrorType = iota
	ErrorTypeCircularDependency
	ErrorTypeUnresolvedReference
	ErrorTypeInvalidSubquery
	ErrorTypeScopeViolation
	ErrorTypeTypeIncompatibility
	ErrorTypeSyntaxError
)

// String returns the string representation of ErrorType
func (et ErrorType) String() string {
	switch et {
	case ErrorTypeCircularDependency:
		return "CircularDependency"
	case ErrorTypeUnresolvedReference:
		return "UnresolvedReference"
	case ErrorTypeInvalidSubquery:
		return "InvalidSubquery"
	case ErrorTypeScopeViolation:
		return "ScopeViolation"
	case ErrorTypeTypeIncompatibility:
		return "TypeIncompatibility"
	case ErrorTypeSyntaxError:
		return "SyntaxError"
	default:
		return "Unknown"
	}
}

// Position represents a position in the source code
type Position struct {
	Line   int    // Line number (1-based)
	Column int    // Column number (1-based)
	Offset int    // Byte offset (0-based)
	File   string // Source file name
}

// String returns a string representation of the position
func (p Position) String() string {
	if p.File != "" {
		return fmt.Sprintf("%s:%d:%d", p.File, p.Line, p.Column)
	}
	return fmt.Sprintf("%d:%d", p.Line, p.Column)
}

// ParseError represents a detailed parsing error
type ParseError struct {
	Type        ErrorType
	Message     string
	Position    Position
	Context     string   // Surrounding context
	Suggestions []string // Suggested fixes
	RelatedIDs  []string // Related node/field IDs
}

// Error implements the error interface
func (pe *ParseError) Error() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("[%s] %s", pe.Type.String(), pe.Message))

	if pe.Position.Line > 0 {
		sb.WriteString(fmt.Sprintf(" at %s", pe.Position.String()))
	}

	if pe.Context != "" {
		sb.WriteString(fmt.Sprintf("\nContext: %s", pe.Context))
	}

	if len(pe.Suggestions) > 0 {
		sb.WriteString("\nSuggestions:")
		for _, suggestion := range pe.Suggestions {
			sb.WriteString(fmt.Sprintf("\n  - %s", suggestion))
		}
	}

	if len(pe.RelatedIDs) > 0 {
		sb.WriteString(fmt.Sprintf("\nRelated: %s", strings.Join(pe.RelatedIDs, ", ")))
	}

	return sb.String()
}

// ErrorReporter collects and manages parsing errors
type ErrorReporter struct {
	errors    []*ParseError
	warnings  []*ParseError
	maxErrors int
}

// NewErrorReporter creates a new error reporter
func NewErrorReporter() *ErrorReporter {
	return &ErrorReporter{
		errors:    make([]*ParseError, 0),
		warnings:  make([]*ParseError, 0),
		maxErrors: 100, // Prevent excessive error collection
	}
}

// AddError adds a new error
func (er *ErrorReporter) AddError(errorType ErrorType, message string, position Position) {
	if len(er.errors) >= er.maxErrors {
		return
	}

	err := &ParseError{
		Type:     errorType,
		Message:  message,
		Position: position,
	}
	er.errors = append(er.errors, err)
}

// AddErrorWithContext adds a new error with context
func (er *ErrorReporter) AddErrorWithContext(errorType ErrorType, message string, position Position, context string, suggestions []string, relatedIDs []string) {
	if len(er.errors) >= er.maxErrors {
		return
	}

	err := &ParseError{
		Type:        errorType,
		Message:     message,
		Position:    position,
		Context:     context,
		Suggestions: suggestions,
		RelatedIDs:  relatedIDs,
	}
	er.errors = append(er.errors, err)
}

// AddWarning adds a new warning
func (er *ErrorReporter) AddWarning(errorType ErrorType, message string, position Position) {
	err := &ParseError{
		Type:     errorType,
		Message:  message,
		Position: position,
	}
	er.warnings = append(er.warnings, err)
}

// HasErrors returns true if there are any errors
func (er *ErrorReporter) HasErrors() bool {
	return len(er.errors) > 0
}

// HasWarnings returns true if there are any warnings
func (er *ErrorReporter) HasWarnings() bool {
	return len(er.warnings) > 0
}

// GetErrors returns all errors
func (er *ErrorReporter) GetErrors() []*ParseError {
	return er.errors
}

// GetWarnings returns all warnings
func (er *ErrorReporter) GetWarnings() []*ParseError {
	return er.warnings
}

// GetErrorsByType returns errors of a specific type
func (er *ErrorReporter) GetErrorsByType(errorType ErrorType) []*ParseError {
	var filtered []*ParseError
	for _, err := range er.errors {
		if err.Type == errorType {
			filtered = append(filtered, err)
		}
	}
	return filtered
}

// Clear removes all errors and warnings
func (er *ErrorReporter) Clear() {
	er.errors = er.errors[:0]
	er.warnings = er.warnings[:0]
}

// String returns a formatted string of all errors and warnings
func (er *ErrorReporter) String() string {
	var sb strings.Builder

	if len(er.errors) > 0 {
		sb.WriteString(fmt.Sprintf("Errors (%d):\n", len(er.errors)))
		for i, err := range er.errors {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, err.Error()))
		}
	}

	if len(er.warnings) > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("Warnings (%d):\n", len(er.warnings)))
		for i, warning := range er.warnings {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, warning.Error()))
		}
	}

	return sb.String()
}

// ErrorCollector is a helper for collecting common error patterns
type ErrorCollector struct {
	reporter *ErrorReporter
}

// NewErrorCollector creates a new error collector
func NewErrorCollector(reporter *ErrorReporter) *ErrorCollector {
	return &ErrorCollector{
		reporter: reporter,
	}
}

// ReportCircularDependency reports a circular dependency error
func (ec *ErrorCollector) ReportCircularDependency(path []string, position Position) {
	message := fmt.Sprintf("Circular dependency detected in path: %s", strings.Join(path, " -> "))
	context := fmt.Sprintf("Dependency chain forms a cycle")
	suggestions := []string{
		"Review the subquery structure to eliminate circular references",
		"Consider restructuring the query to avoid circular dependencies",
	}

	ec.reporter.AddErrorWithContext(
		ErrorTypeCircularDependency,
		message,
		position,
		context,
		suggestions,
		path,
	)
}

// ReportUnresolvedReference reports an unresolved reference error
func (ec *ErrorCollector) ReportUnresolvedReference(fieldName string, availableFields []string, position Position) {
	message := fmt.Sprintf("Unresolved field reference: '%s'", fieldName)
	context := fmt.Sprintf("Field '%s' is not available in current scope", fieldName)

	var suggestions []string
	if len(availableFields) > 0 {
		suggestions = append(suggestions, "Available fields in current scope:")
		for _, field := range availableFields {
			suggestions = append(suggestions, fmt.Sprintf("  - %s", field))
		}
	}
	suggestions = append(suggestions, "Check the field name for typos")
	suggestions = append(suggestions, "Ensure the field is defined in an accessible scope")

	ec.reporter.AddErrorWithContext(
		ErrorTypeUnresolvedReference,
		message,
		position,
		context,
		suggestions,
		[]string{fieldName},
	)
}

// ReportScopeViolation reports a scope violation error
func (ec *ErrorCollector) ReportScopeViolation(fieldName string, currentScope string, requiredScope string, position Position) {
	message := fmt.Sprintf("Scope violation: field '%s' not accessible from %s", fieldName, currentScope)
	context := fmt.Sprintf("Field requires scope '%s' but current scope is '%s'", requiredScope, currentScope)
	suggestions := []string{
		fmt.Sprintf("Move the reference to %s scope", requiredScope),
		"Review the query structure to ensure proper scoping",
	}

	ec.reporter.AddErrorWithContext(
		ErrorTypeScopeViolation,
		message,
		position,
		context,
		suggestions,
		[]string{fieldName},
	)
}

// ReportInvalidSubquery reports an invalid subquery error
func (ec *ErrorCollector) ReportInvalidSubquery(subqueryID string, reason string, position Position) {
	message := fmt.Sprintf("Invalid subquery '%s': %s", subqueryID, reason)
	context := fmt.Sprintf("Subquery validation failed")
	suggestions := []string{
		"Check the subquery syntax",
		"Ensure all referenced fields are available",
		"Verify the subquery structure matches expected format",
	}

	ec.reporter.AddErrorWithContext(
		ErrorTypeInvalidSubquery,
		message,
		position,
		context,
		suggestions,
		[]string{subqueryID},
	)
}
