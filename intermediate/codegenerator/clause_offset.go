package codegenerator

import (
	"github.com/shibukawa/snapsql/parser"
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
		builder.addIfSystemOffset()
		builder.addStatic(" OFFSET ", nil)
		builder.addEmitSystemOffset()
		builder.addEndCondition(nil)

		return nil
	}

	// OFFSET clause is present in SQL
	tokens := offsetClause.RawTokens()

	// Output "OFFSET" keyword with space
	builder.addStatic(" OFFSET ", &tokens[0].Position)

	// Register system OFFSET block with default value
	// This creates: IF_SYSTEM_OFFSET { emit system offset } with default fallback
	builder.addIfSystemOffset()
	builder.addEmitSystemOffset()
	builder.addRawElseCondition(nil)
	err := builder.ProcessTokens(tokens[2:])
	builder.addEndCondition(nil)

	return err
}
