package codegenerator

import (
	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
)

// GenerateForClauseOrSystem generates instructions for the FOR clause or system FOR if not present.
//
// This is a unified function that handles both cases:
//
// 1. **FOR clause present in SQL**:
//   - Processes tokens through ProcessTokens() to handle directives and static text
//   - Example: FOR UPDATE, FOR SHARE, etc.
//
// 2. **FOR clause NOT present** (forClause == nil):
//   - Calls RegisterEmitSystemFor() to emit EMIT_SYSTEM_FOR instruction
//   - This allows system-provided FOR clause (e.g., FOR UPDATE) to be output at runtime
//
// The caller passes nil when the FOR clause is not present, and this function
// handles both cases transparently.
func GenerateForClauseOrSystem(forClause *parser.ForClause, builder *InstructionBuilder) error {
	if forClause == nil {
		// FOR clause is not present in SQL.
		// SQLite 方言では悲観ロック構文を生成しないため EMIT_SYSTEM_FOR をスキップする。
		if builder != nil && builder.context != nil && builder.context.Dialect == snapsql.DialectSQLite {
			return nil
		}

		// Emit system FOR if provided at runtime
		builder.RegisterEmitSystemFor()

		return nil
	}

	// FOR clause is present in SQL
	// Process tokens (may contain directives and static text)
	allTokens := forClause.RawTokens()
	builder.addStatic(" ", &allTokens[0].Position)

	return builder.ProcessTokens(allTokens)
}
