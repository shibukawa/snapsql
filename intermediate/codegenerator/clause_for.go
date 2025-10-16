package codegenerator

import (
	"fmt"

	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/tokenizer"
)

// generateForClause generates instructions for the FOR clause (row-lock)
// when the FOR clause is present in the SQL statement.
//
// For example:
//
//	SQL: SELECT id FROM users FOR UPDATE
//	Output: EMIT_STATIC(" FOR UPDATE")
//
// When FOR clause is present, it's simply emitted as static text.
func generateForClause(clause parser.ClauseNode, builder *InstructionBuilder) error {
	// Get all tokens (heading + body) to include the FOR keyword
	forClause, ok := clause.(*parser.ForClause)
	if !ok {
		return fmt.Errorf("%w: expected *parser.ForClause, got %T", ErrStatementTypeMismatch, clause)
	}

	allTokens := forClause.RawTokens()

	// Extract FOR clause value from tokens
	forClauseValue := extractForClauseValue(allTokens)

	// Get position from first token
	pos := ""
	if len(allTokens) > 0 {
		pos = formatPosition(allTokens[0].Position)
	}

	// Simply emit the FOR clause as static text
	builder.AddInstruction(Instruction{
		Op:    OpEmitStatic,
		Value: forClauseValue,
		Pos:   pos,
	})

	return nil
}

// extractForClauseValue extracts the FOR clause string from tokens.
// The SourceText() method provides the complete FOR clause string directly

// extractForClauseValue extracts the FOR clause string from tokens.
// Example: tokens ["FOR", " ", "UPDATE", " ", "NOWAIT"] -> " FOR UPDATE NOWAIT"
func extractForClauseValue(tokens []tokenizer.Token) string {
	if len(tokens) == 0 {
		return ""
	}

	result := ""

	for i, token := range tokens {
		// Skip directive tokens
		if token.Directive != nil {
			continue
		}

		// Add token value
		result += token.Value

		// Add space after non-whitespace tokens if next token is non-whitespace
		// (whitespace tokens should preserve their own spacing)
		if i < len(tokens)-1 {
			nextToken := tokens[i+1]
			// If current token is not whitespace and next token is not whitespace,
			// add a space between them
			if token.Type != tokenizer.WHITESPACE && nextToken.Type != tokenizer.WHITESPACE {
				// Check if result doesn't already end with space
				if len(result) > 0 && result[len(result)-1] != ' ' {
					result += " "
				}
			}
		}
	}

	// Ensure it starts with a space
	if result != "" && result[0] != ' ' {
		result = " " + result
	}

	return result
}

// GenerateSystemForIfNotExists generates system FOR clause instruction
// when the FOR clause is NOT present in the SQL statement.
//
// Output: EMIT_SYSTEM_FOR
//
// This allows the FOR clause to be added at runtime if needed.
// The instruction is not wrapped in IF - it's always present.
func GenerateSystemForIfNotExists(builder *InstructionBuilder) {
	// Add EMIT_SYSTEM_FOR - output the system FOR clause value if provided
	builder.AddInstruction(Instruction{
		Op:  OpEmitSystemFor,
		Pos: "",
	})
}
