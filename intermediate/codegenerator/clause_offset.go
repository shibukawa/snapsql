package codegenerator

import (
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/tokenizer"
)

// GenerateOffsetClauseOrSystem generates instructions for the OFFSET clause or system OFFSET if not present.
//
// This is a unified function that handles both cases:
//
// 1. **OFFSET clause present in SQL**:
//   - Extracts the default offset value
//   - Registers IF_SYSTEM_OFFSET instruction to allow runtime override
//   - Emits the system offset value (with fallback to default value)
//
// 2. **OFFSET clause NOT present** (offsetClause == nil):
//   - Registers IF_SYSTEM_OFFSET instruction to conditionally emit OFFSET keyword and value
//   - Allows system-provided OFFSET to be output at runtime
//
// The caller passes nil when the OFFSET clause is not present, and this function
// handles both cases transparently.
func GenerateOffsetClauseOrSystem(offsetClause *parser.OffsetClause, builder *InstructionBuilder) error {
	if offsetClause == nil {
		// OFFSET clause is not present in SQL
		// Conditionally emit OFFSET keyword and system value if provided at runtime
		builder.RegisterIfSystemOffset("", "")
		builder.RegisterEmitStatic(" OFFSET ", "")
		builder.RegisterEmitSystemOffset()

		return nil
	}

	// OFFSET clause is present in SQL
	tokens := offsetClause.RawTokens()

	// Extract OFFSET literal value for default
	var defaultValue string

	for _, token := range tokens {
		if token.Type == tokenizer.NUMBER {
			defaultValue = token.Value
			break
		}
	}

	// Output "OFFSET" keyword with space
	builder.RegisterEmitStatic(" OFFSET ", "")

	// Register system OFFSET block with default value
	// This creates: IF_SYSTEM_OFFSET { emit system offset } with default fallback
	builder.RegisterIfSystemOffset(defaultValue, "")
	builder.RegisterEmitSystemOffset()

	return nil
}
