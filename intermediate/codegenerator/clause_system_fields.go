package codegenerator

import (
	"fmt"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/tokenizer"
)

// findClosingParenIndex finds the index of the closing parenthesis in a token slice.
// Returns the index of the token containing ')', or -1 if not found.
// Handles nested parentheses by counting depth.
func findClosingParenIndex(tokens []tokenizer.Token) int {
	parenDepth := 0
	for i, token := range tokens {
		for _, ch := range token.Value {
			if ch == '(' {
				parenDepth++
			} else if ch == ')' {
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
func insertSystemFieldValues(builder *InstructionBuilder, fields []snapsql.SystemField) {
	if len(fields) == 0 {
		return
	}

	for _, field := range fields {
		// Add comma separator
		builder.instructions = append(builder.instructions, Instruction{
			Op:    OpEmitStatic,
			Value: ", ",
		})

		// Add system field value as OpEmitSystemValue instruction
		builder.instructions = append(builder.instructions, Instruction{
			Op:           OpEmitSystemValue,
			SystemField:  field.Name,
			DefaultValue: fmt.Sprintf("%v", field.OnInsert.Default),
		})
	}
}

// appendSystemFieldUpdates appends system field updates to SET clause.
// Adds OpEmitStatic with ", field = default" followed by OpEmitSystemValue instructions.
func appendSystemFieldUpdates(builder *InstructionBuilder, fields []snapsql.SystemField) {
	if len(fields) == 0 {
		return
	}

	for _, field := range fields {
		// Add comma and field assignment
		builder.instructions = append(builder.instructions, Instruction{
			Op:    OpEmitStatic,
			Value: ", " + field.Name + " = ",
		})

		// Add system field value as OpEmitSystemValue instruction
		builder.instructions = append(builder.instructions, Instruction{
			Op:           OpEmitSystemValue,
			SystemField:  field.Name,
			DefaultValue: fmt.Sprintf("%v", field.OnUpdate.Default),
		})
	}
}
