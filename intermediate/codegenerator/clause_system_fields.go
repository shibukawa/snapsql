package codegenerator

import (
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/tokenizer"
)

// normalizeDefaultExpressionForDialect converts default expressions based on the SQL dialect.
// For example, NOW() in PostgreSQL becomes CURRENT_TIMESTAMP in SQLite.
func normalizeDefaultExpressionForDialect(expr interface{}, dialect snapsql.Dialect) string {
	exprStr := fmt.Sprintf("%v", expr)
	exprUpper := strings.ToUpper(strings.TrimSpace(exprStr))

	// Normalize NOW() variants
	if exprUpper == "NOW()" || exprUpper == "NOW" {
		switch dialect {
		case snapsql.DialectSQLite:
			return "CURRENT_TIMESTAMP"
		case snapsql.DialectMySQL:
			return "NOW()"
		case snapsql.DialectMariaDB:
			return "NOW()"
		case snapsql.DialectPostgres:
			return "NOW()"
		default:
			return "CURRENT_TIMESTAMP"
		}
	}

	// Normalize CURRENT_TIMESTAMP variants
	if exprUpper == "CURRENT_TIMESTAMP" {
		return "CURRENT_TIMESTAMP"
	}

	// For other expressions, return as-is
	return exprStr
}

// findClosingParenIndex finds the index of the closing parenthesis in a token slice.
// Returns the index of the token containing ')', or -1 if not found.
// Handles nested parentheses by counting depth.
func findClosingParenIndex(tokens []tokenizer.Token) int {
	parenDepth := 0

	for i, token := range tokens {
		for _, ch := range token.Value {
			switch ch {
			case '(':
				parenDepth++
			case ')':
				parenDepth--
				if parenDepth == 0 {
					return i
				}
			}
		}
	}

	return -1
}

// getInsertSystemFields returns list of system fields that should be inserted.
// Only fields with OnInsert.Default or OnInsert.Parameter are included.
func getInsertSystemFields(ctx *GenerationContext) []snapsql.SystemField {
	if ctx.Config == nil || len(ctx.Config.System.Fields) == 0 {
		return nil
	}

	var fields []snapsql.SystemField

	for _, field := range ctx.Config.System.Fields {
		if field.OnInsert.Default != nil || field.OnInsert.Parameter != "" {
			fields = append(fields, field)
		}
	}

	return fields
}

// getInsertSystemFieldsFiltered returns list of system fields to be inserted,
// excluding fields that already exist in the provided column names.
// This prevents duplicate field names in the column list.
func getInsertSystemFieldsFiltered(ctx *GenerationContext, existingColumns map[string]bool) []snapsql.SystemField {
	fields := getInsertSystemFields(ctx)
	if len(fields) == 0 {
		return nil
	}

	var filtered []snapsql.SystemField

	for _, field := range fields {
		// Only include if not already in the column list
		if !existingColumns[field.Name] {
			filtered = append(filtered, field)
		}
	}

	return filtered
}

// getUpdateSystemFields returns list of system fields that should be updated.
// Only fields with OnUpdate.Default or OnUpdate.Parameter are included.
func getUpdateSystemFields(ctx *GenerationContext) []snapsql.SystemField {
	if ctx.Config == nil || len(ctx.Config.System.Fields) == 0 {
		return nil
	}

	var fields []snapsql.SystemField

	for _, field := range ctx.Config.System.Fields {
		if field.OnUpdate.Default != nil || field.OnUpdate.Parameter != "" {
			fields = append(fields, field)
		}
	}

	return fields
}

// getUpdateSystemFieldsFiltered returns list of system fields to be updated,
// excluding fields that already exist in the provided SET clause field names.
// This prevents duplicate field names in the SET clause.
func getUpdateSystemFieldsFiltered(ctx *GenerationContext, existingFields map[string]bool) []snapsql.SystemField {
	fields := getUpdateSystemFields(ctx)
	if len(fields) == 0 {
		return nil
	}

	var filtered []snapsql.SystemField

	for _, field := range fields {
		// Only include if not already in the SET clause
		if !existingFields[field.Name] {
			filtered = append(filtered, field)
		}
	}

	return filtered
}

// insertSystemFieldNames adds system field names to the instruction builder.
// Adds field names as EMIT_STATIC instructions with comma separators.
func insertSystemFieldNames(builder *InstructionBuilder, fields []snapsql.SystemField) {
	if len(fields) == 0 {
		return
	}

	for _, field := range fields {
		// Add comma and field name as separate instruction
		builder.instructions = append(builder.instructions, Instruction{
			Op:    OpEmitStatic,
			Value: ", " + field.Name,
		})
	}
}

// insertSystemFieldValues appends system field values to VALUES clause.
// Inserts OpEmitSystemValue instructions for each system field.
// Default values are normalized based on SQL dialect.
// Only inserts once - subsequent calls are no-ops.
func insertSystemFieldValues(builder *InstructionBuilder, fields []snapsql.SystemField) {
	if len(fields) == 0 || builder.systemFieldsAdded {
		return
	}

	// Mark that system fields have been added
	builder.systemFieldsAdded = true

	// Get dialect from builder context
	dialect := builder.context.Dialect

	for _, field := range fields {
		// Add comma separator
		builder.instructions = append(builder.instructions, Instruction{
			Op:    OpEmitStatic,
			Value: ", ",
		})

		// Normalize default expression based on dialect
		defaultValue := normalizeDefaultExpressionForDialect(field.OnInsert.Default, dialect)

		// Add system field value as OpEmitSystemValue instruction
		builder.instructions = append(builder.instructions, Instruction{
			Op:           OpEmitSystemValue,
			SystemField:  field.Name,
			DefaultValue: defaultValue,
		})
	}
}

// appendSystemFieldUpdates appends system field updates to SET clause.
// Adds OpEmitStatic with ", field = default" followed by OpEmitSystemValue instructions.
// Default values are normalized based on SQL dialect.
func appendSystemFieldUpdates(builder *InstructionBuilder, fields []snapsql.SystemField) {
	if len(fields) == 0 {
		return
	}

	// Get dialect from builder context
	dialect := builder.context.Dialect

	for _, field := range fields {
		// Add comma and field assignment
		builder.instructions = append(builder.instructions, Instruction{
			Op:    OpEmitStatic,
			Value: ", " + field.Name + " = ",
		})

		// Normalize default expression based on dialect
		defaultValue := normalizeDefaultExpressionForDialect(field.OnUpdate.Default, dialect)

		// Add system field value as OpEmitSystemValue instruction
		builder.instructions = append(builder.instructions, Instruction{
			Op:           OpEmitSystemValue,
			SystemField:  field.Name,
			DefaultValue: defaultValue,
		})
	}
}

// appendSystemFieldsToSelectClause appends system field expressions to SELECT clause for INSERT...SELECT.
// For each system field with OnInsert.Default or OnInsert.Parameter, adds ", default_expr AS field_name"
// to the SELECT clause. Default expressions are normalized based on SQL dialect.
func appendSystemFieldsToSelectClause(builder *InstructionBuilder, fields []snapsql.SystemField) {
	if len(fields) == 0 {
		return
	}

	// Get dialect from builder context
	dialect := builder.context.Dialect

	for i, field := range fields {
		// Add comma separator
		builder.instructions = append(builder.instructions, Instruction{
			Op:    OpEmitStatic,
			Value: ", ",
		})

		// Determine the expression to use for this field
		var expr string
		if field.OnInsert.Default != nil {
			// Normalize default expression based on dialect
			expr = normalizeDefaultExpressionForDialect(field.OnInsert.Default, dialect)
		} else if field.OnInsert.Parameter != "" && field.OnInsert.Parameter != snapsql.ParameterExplicit {
			// For non-explicit parameters, use the parameter directly
			// This would be handled via CEL variable references
			expr = string(field.OnInsert.Parameter)
		}

		// Add field expression as static emit
		if expr != "" {
			value := expr + " AS " + field.Name
			// Add space after the last field to separate from FROM clause
			if i == len(fields)-1 {
				value += " "
			}

			builder.instructions = append(builder.instructions, Instruction{
				Op:    OpEmitStatic,
				Value: value,
			})
		}
	}
}

// validateExplicitSystemFields validates that explicit system fields have corresponding parameters.
// For INSERT: checks OnInsert.Parameter == "explicit"
// For UPDATE: checks OnUpdate.Parameter == "explicit"
func validateExplicitSystemFields(ctx *GenerationContext, fields []snapsql.SystemField, operation string) error {
	if len(fields) == 0 {
		return nil
	}

	// Get all available parameters from function definition
	availableParams := make(map[string]bool)

	if ctx.FunctionDefinition != nil && ctx.FunctionDefinition.ParameterOrder != nil {
		for _, paramName := range ctx.FunctionDefinition.ParameterOrder {
			availableParams[paramName] = true
		}
	}

	// Check each field
	for _, field := range fields {
		var paramType string
		switch operation {
		case "insert":
			paramType = string(field.OnInsert.Parameter)
		case "update":
			paramType = string(field.OnUpdate.Parameter)
		}

		// If parameter is "explicit", it must be provided in function parameters
		if paramType == "explicit" {
			if !availableParams[field.Name] {
				return fmt.Errorf(
					"system field '%s' requires explicit parameter '%s' for %s operation, but parameter is not defined in function",
					field.Name, field.Name, operation,
				)
			}
		}
	}

	return nil
}
