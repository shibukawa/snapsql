package codegenerator

import (
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/tokenizer"
)

// GenerateLimitClauseOrSystem generates instructions for the LIMIT clause or system LIMIT if not present.
//
// This is a unified function that handles both cases:
//
// 1. **LIMIT clause present in SQL**:
//   - Extracts the default limit value
//   - Registers IF_SYSTEM_LIMIT instruction to allow runtime override
//   - Emits the system limit value (with fallback to default value)
//
// 2. **LIMIT clause NOT present** (limitClause == nil):
//   - Registers IF_SYSTEM_LIMIT instruction to conditionally emit LIMIT keyword and value
//   - Allows system-provided LIMIT to be output at runtime
//
// The caller passes nil when the LIMIT clause is not present, and this function
// handles both cases transparently.
func GenerateLimitClauseOrSystem(limitClause *parser.LimitClause, builder *InstructionBuilder) error {
	if limitClause == nil {
		// LIMIT clause is not present in SQL
		// Conditionally emit LIMIT keyword and system value if provided at runtime
		builder.RegisterIfSystemLimit("", "")
		builder.RegisterEmitStatic(" LIMIT ", "")
		builder.RegisterEmitSystemLimit()

		return nil
	}

	// LIMIT clause is present in SQL
	tokens := limitClause.RawTokens()

	// Extract LIMIT literal value for default
	var defaultValue string

	for _, token := range tokens {
		if token.Type == tokenizer.NUMBER {
			defaultValue = token.Value
			break
		}
	}

	// Output "LIMIT" keyword with space
	builder.RegisterEmitStatic(" LIMIT ", "")

	// Register system LIMIT block with default value
	// This creates: IF_SYSTEM_LIMIT { emit system limit } with default fallback
	builder.RegisterIfSystemLimit(defaultValue, "")
	builder.RegisterEmitSystemLimit()

	return nil
}
