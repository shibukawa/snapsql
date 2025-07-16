package typeinference

import (
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser/parsercommon"
)

// ValidationErrorType represents the type of validation error
type ValidationErrorType int

const (
	TableNotFound ValidationErrorType = iota
	ColumnNotFound
	AmbiguousColumn
	SchemaNotFound
	TypeMismatch
	NullabilityViolation
	CircularReference
	ExpressionTooComplex
)

// ErrorSeverity represents the severity of an error
type ErrorSeverity int

const (
	Warning ErrorSeverity = iota // Warning: inference failed, continue with 'any' type
	Error                        // Error: stop type inference
)

// ValidationError represents a schema validation error
type ValidationError struct {
	FieldIndex int                 // Index of the problematic field
	ErrorType  ValidationErrorType // Type of error
	Message    string              // Error message
	Severity   ErrorSeverity       // Error severity
	TableName  string              // Related table name (for schema errors)
	ColumnName string              // Related column name (for schema errors)
	Suggestion string              // Correction suggestion
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return e.Message
}

// TypeInferenceError represents a type inference error
type TypeInferenceError struct {
	FieldIndex int                 // Index of the problematic field
	ErrorType  ValidationErrorType // Type of error
	Message    string              // Error message
	Severity   ErrorSeverity       // Error severity
	TableName  string              // Related table name (for schema errors)
	ColumnName string              // Related column name (for schema errors)
}

// Error implements the error interface
func (e *TypeInferenceError) Error() string {
	return e.Message
}

// SchemaValidator validates SQL statements against database schema
type SchemaValidator struct {
	schemaResolver  *SchemaResolver
	tableAliases    map[string]string // alias -> real table name
	availableTables []string          // Currently available tables
	errors          []ValidationError
	warnings        []ValidationError
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator(schemaResolver *SchemaResolver) *SchemaValidator {
	return &SchemaValidator{
		schemaResolver:  schemaResolver,
		tableAliases:    make(map[string]string),
		availableTables: []string{},
		errors:          []ValidationError{},
		warnings:        []ValidationError{},
	}
}

// SetTableAliases sets the table alias mappings
func (v *SchemaValidator) SetTableAliases(aliases map[string]string) {
	v.tableAliases = aliases
}

// SetAvailableTables sets the currently available tables
func (v *SchemaValidator) SetAvailableTables(tables []string) {
	v.availableTables = tables
}

// ValidateSelectFields validates SELECT fields against schema
func (v *SchemaValidator) ValidateSelectFields(selectFields []parsercommon.SelectField) []ValidationError {
	v.errors = []ValidationError{}
	v.warnings = []ValidationError{}

	for i, field := range selectFields {
		switch field.FieldKind {
		case parsercommon.TableField:
			if err := v.validateTableColumn(i, field.TableName, field.OriginalField); err != nil {
				v.errors = append(v.errors, *err)
			}

		case parsercommon.SingleField:
			if err := v.validateSingleColumn(i, field.OriginalField); err != nil {
				v.errors = append(v.errors, *err)
			}

		case parsercommon.FunctionField, parsercommon.ComplexField, parsercommon.LiteralField:
			// These field types don't require schema validation
			continue
		}
	}

	return append(v.errors, v.warnings...)
}

// validateTableColumn validates a specific table.column reference
func (v *SchemaValidator) validateTableColumn(fieldIndex int, tableName, columnName string) *ValidationError {
	// Resolve table alias
	realTableName := v.resolveTableAlias(tableName)

	// Find schema name for the table
	schemaName := v.findSchemaForTable(realTableName)
	if schemaName == "" {
		return &ValidationError{
			FieldIndex: fieldIndex,
			ErrorType:  TableNotFound,
			Message:    fmt.Sprintf("Table '%s' does not exist", realTableName),
			Severity:   Error,
			TableName:  realTableName,
			Suggestion: v.suggestSimilarTables(realTableName),
		}
	}

	// Validate column exists
	_, err := v.schemaResolver.ResolveTableColumn(schemaName, realTableName, columnName)
	if err != nil {
		return &ValidationError{
			FieldIndex: fieldIndex,
			ErrorType:  ColumnNotFound,
			Message:    fmt.Sprintf("Column '%s' does not exist in table '%s.%s'", columnName, schemaName, realTableName),
			Severity:   Error,
			TableName:  realTableName,
			ColumnName: columnName,
			Suggestion: v.suggestSimilarColumns(schemaName, realTableName, columnName),
		}
	}

	return nil
}

// validateSingleColumn validates a column reference without table qualification
func (v *SchemaValidator) validateSingleColumn(fieldIndex int, columnName string) *ValidationError {
	// Find all tables that contain this column
	matches := v.schemaResolver.FindColumnInTables(columnName, v.availableTables)

	if len(matches) == 0 {
		return &ValidationError{
			FieldIndex: fieldIndex,
			ErrorType:  ColumnNotFound,
			Message:    fmt.Sprintf("Column '%s' does not exist in any available table", columnName),
			Severity:   Error,
			ColumnName: columnName,
			Suggestion: v.suggestSimilarColumnsAcrossTables(columnName),
		}
	}

	if len(matches) > 1 {
		return &ValidationError{
			FieldIndex: fieldIndex,
			ErrorType:  AmbiguousColumn,
			Message:    fmt.Sprintf("Column '%s' is ambiguous, found in tables: %v", columnName, matches),
			Severity:   Error,
			ColumnName: columnName,
			Suggestion: "Use table.column notation to disambiguate",
		}
	}

	return nil
}

// resolveTableAlias resolves a table alias to the real table name
func (v *SchemaValidator) resolveTableAlias(tableName string) string {
	if realName, exists := v.tableAliases[tableName]; exists {
		return realName
	}
	return tableName
}

// findSchemaForTable finds the schema name that contains the given table
func (v *SchemaValidator) findSchemaForTable(tableName string) string {
	for _, schemaName := range v.schemaResolver.GetAllSchemas() {
		tables := v.schemaResolver.GetTablesInSchema(schemaName)
		for _, table := range tables {
			if table == tableName {
				return schemaName
			}
		}
	}
	return ""
}

// ValidateTypeConsistency validates type consistency between inferred and schema types
func (v *SchemaValidator) ValidateTypeConsistency(
	field *parsercommon.SelectField,
	inferredType *TypeInfo,
	schemaColumn *snapsql.ColumnInfo,
) *ValidationError {
	// CAST expressions are always allowed (explicit type conversion)
	if field.ExplicitType {
		return nil
	}

	// Check basic type compatibility
	if !v.areTypesCompatible(inferredType.BaseType, schemaColumn.DataType) {
		return &ValidationError{
			ErrorType: TypeMismatch,
			Message: fmt.Sprintf("Type mismatch: inferred '%s' but schema expects '%s'",
				inferredType.BaseType, schemaColumn.DataType),
			Severity: Warning, // Warning level for type mismatches
		}
	}

	// Check nullability
	if !inferredType.IsNullable && schemaColumn.Nullable {
		return &ValidationError{
			ErrorType: NullabilityViolation,
			Message: fmt.Sprintf("Column '%s' is nullable in schema but inferred as non-nullable",
				field.OriginalField),
			Severity: Warning, // Warning level for nullability issues
		}
	}

	return nil
}

// areTypesCompatible checks if two types are compatible
func (v *SchemaValidator) areTypesCompatible(inferredType, schemaType string) bool {
	// Normalize types using the function from types.go
	normalizedInferred := normalizeType(inferredType)
	normalizedSchema := normalizeType(schemaType)

	// Exact match
	if normalizedInferred == normalizedSchema {
		return true
	}

	// Compatible type combinations
	compatiblePairs := map[string][]string{
		"int":       {"bigint", "smallint", "decimal", "float"},
		"float":     {"double", "decimal", "int"},
		"string":    {"text", "varchar", "char"},
		"timestamp": {"datetime", "date"},
		"decimal":   {"numeric", "float", "int"},
	}

	if compatible, exists := compatiblePairs[normalizedInferred]; exists {
		for _, compatType := range compatible {
			if compatType == normalizedSchema {
				return true
			}
		}
	}

	return false
}

// Suggestion methods for error correction

// suggestSimilarTables suggests similar table names
func (v *SchemaValidator) suggestSimilarTables(target string) string {
	var suggestions []string
	minDistance := 3

	for _, schemaName := range v.schemaResolver.GetAllSchemas() {
		tables := v.schemaResolver.GetTablesInSchema(schemaName)
		for _, tableName := range tables {
			distance := levenshteinDistance(target, tableName)
			if distance <= minDistance && distance > 0 {
				suggestions = append(suggestions, fmt.Sprintf("%s.%s", schemaName, tableName))
			}
		}
	}

	if len(suggestions) > 0 {
		return fmt.Sprintf("Did you mean: %s", strings.Join(suggestions[:min(3, len(suggestions))], ", "))
	}

	return "No similar tables found"
}

// suggestSimilarColumns suggests similar column names within a table
func (v *SchemaValidator) suggestSimilarColumns(schemaName, tableName, target string) string {
	columns, err := v.schemaResolver.GetTableColumns(schemaName, tableName)
	if err != nil {
		return "No suggestions available"
	}

	var suggestions []string
	minDistance := 3

	for _, column := range columns {
		distance := levenshteinDistance(target, column.Name)
		if distance <= minDistance && distance > 0 {
			suggestions = append(suggestions, column.Name)
		}
	}

	if len(suggestions) > 0 {
		return fmt.Sprintf("Did you mean: %s", strings.Join(suggestions[:min(3, len(suggestions))], ", "))
	}

	return "No similar columns found"
}

// suggestSimilarColumnsAcrossTables suggests similar column names across all available tables
func (v *SchemaValidator) suggestSimilarColumnsAcrossTables(target string) string {
	var suggestions []string
	minDistance := 3

	for _, schemaName := range v.schemaResolver.GetAllSchemas() {
		tables := v.schemaResolver.GetTablesInSchema(schemaName)
		for _, tableName := range tables {
			columns, err := v.schemaResolver.GetTableColumns(schemaName, tableName)
			if err != nil {
				continue
			}

			for _, column := range columns {
				distance := levenshteinDistance(target, column.Name)
				if distance <= minDistance && distance > 0 {
					suggestions = append(suggestions, fmt.Sprintf("%s.%s.%s", schemaName, tableName, column.Name))
				}
			}
		}
	}

	if len(suggestions) > 0 {
		return fmt.Sprintf("Did you mean: %s", strings.Join(suggestions[:min(3, len(suggestions))], ", "))
	}

	return "No similar columns found"
}

// Helper functions

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	matrix := make([][]int, len(a)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(b)+1)
	}

	for i := 0; i <= len(a); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(b); j++ {
		matrix[0][j] = j
	}

	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1, // deletion
				min(
					matrix[i][j-1]+1,      // insertion
					matrix[i-1][j-1]+cost, // substitution
				),
			)
		}
	}

	return matrix[len(a)][len(b)]
}
