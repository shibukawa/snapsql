package typeinference

import (
	"fmt"
	"slices"
	"strings"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
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

// String returns the string representation of ValidationErrorType
func (v ValidationErrorType) String() string {
	switch v {
	case TableNotFound:
		return "table_not_found"
	case ColumnNotFound:
		return "column_not_found"
	case AmbiguousColumn:
		return "ambiguous_column"
	case SchemaNotFound:
		return "schema_not_found"
	case TypeMismatch:
		return "type_mismatch"
	case NullabilityViolation:
		return "nullability_violation"
	case CircularReference:
		return "circular_reference"
	case ExpressionTooComplex:
		return "expression_too_complex"
	default:
		return "unknown_error"
	}
}

// ErrorSeverity represents the severity of an error
type ErrorSeverity int

const (
	Warning ErrorSeverity = iota // Warning: inference failed, continue with 'any' type
	Error                        // Error: stop type inference
)

// SchemaValidator validates SQL statements against database schema
type SchemaValidator struct {
	schemaResolver  *SchemaResolver
	tableAliases    map[string]string // alias -> real table name
	availableTables []string          // Currently available tables
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator(schemaResolver *SchemaResolver) *SchemaValidator {
	if schemaResolver == nil {
		return nil
	}

	return &SchemaValidator{
		schemaResolver:  schemaResolver,
		tableAliases:    make(map[string]string),
		availableTables: []string{},
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
func (v *SchemaValidator) ValidateSelectFields(selectFields []parser.SelectField) []ValidationError {
	errors := []ValidationError{}

	for i, field := range selectFields {
		switch field.FieldKind {
		case parser.TableField:
			err := v.validateTableColumn(i, field.TableName, field.OriginalField)
			if err != nil {
				errors = append(errors, *err)
			}

		case parser.SingleField:
			err := v.validateSingleColumn(i, field.OriginalField)
			if err != nil {
				errors = append(errors, *err)
			}

		case parser.FunctionField, parser.ComplexField, parser.LiteralField:
			// These field types don't require schema validation
			continue
		}
	}

	return errors
}

// validateTableColumn validates a specific table.column reference
func (v *SchemaValidator) validateTableColumn(fieldIndex int, tableName, columnName string) *ValidationError {
	// Resolve table alias
	realTableName := v.resolveTableAlias(tableName)

	// Extract column name from table.column format if needed
	realColumnName := columnName

	if strings.Contains(columnName, ".") {
		parts := strings.Split(columnName, ".")
		if len(parts) == 2 {
			realColumnName = parts[1] // Take the column part
		}
	}

	// Find schema name for the table
	schemaName := v.findSchemaForTable(realTableName)
	if schemaName == "" {
		return &ValidationError{
			Position:   fieldIndex,
			ErrorType:  TableNotFound.String(),
			Message:    fmt.Sprintf("Table '%s' does not exist", realTableName),
			TableName:  realTableName,
			Suggestion: v.suggestSimilarTables(realTableName),
		}
	}

	// Validate column exists
	_, err := v.schemaResolver.ResolveTableColumn(schemaName, realTableName, realColumnName)
	if err != nil {
		return &ValidationError{
			Position:   fieldIndex,
			ErrorType:  ColumnNotFound.String(),
			Message:    fmt.Sprintf("Column '%s' does not exist in table '%s.%s'", realColumnName, schemaName, realTableName),
			TableName:  realTableName,
			FieldName:  realColumnName,
			Suggestion: v.suggestSimilarColumns(schemaName, realTableName, realColumnName),
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
			Position:   fieldIndex,
			ErrorType:  ColumnNotFound.String(),
			Message:    fmt.Sprintf("Column '%s' does not exist in any available table", columnName),
			FieldName:  columnName,
			Suggestion: v.suggestSimilarColumnsAcrossTables(columnName),
		}
	}

	if len(matches) > 1 {
		return &ValidationError{
			Position:   fieldIndex,
			ErrorType:  AmbiguousColumn.String(),
			Message:    fmt.Sprintf("Column '%s' is ambiguous, found in tables: %v", columnName, matches),
			FieldName:  columnName,
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
		if slices.Contains(tables, tableName) {
			return schemaName
		}
	}

	return ""
}

// ValidateTypeConsistency validates type consistency between inferred and schema types
func (v *SchemaValidator) ValidateTypeConsistency(
	field *parser.SelectField,
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
			ErrorType: TypeMismatch.String(),
			Message: fmt.Sprintf("Type mismatch: inferred '%s' but schema expects '%s'",
				inferredType.BaseType, schemaColumn.DataType),
		}
	}

	// Check nullability
	if !inferredType.IsNullable && schemaColumn.Nullable {
		return &ValidationError{
			ErrorType: NullabilityViolation.String(),
			Message: fmt.Sprintf("Column '%s' is nullable in schema but inferred as non-nullable",
				field.OriginalField),
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
		if slices.Contains(compatible, normalizedSchema) {
			return true
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
		return "Did you mean: " + strings.Join(suggestions[:minInt(3, len(suggestions))], ", ")
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
		return "Did you mean: " + strings.Join(suggestions[:minInt(3, len(suggestions))], ", ")
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
		return "Did you mean: " + strings.Join(suggestions[:minInt(3, len(suggestions))], ", ")
	}

	return "No similar columns found"
}

// Helper functions

// minInt returns the minimum of two integers
func minInt(a, b int) int {
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

			matrix[i][j] = minInt(
				matrix[i-1][j]+1, // deletion
				minInt(
					matrix[i][j-1]+1,      // insertion
					matrix[i-1][j-1]+cost, // substitution
				),
			)
		}
	}

	return matrix[len(a)][len(b)]
}
