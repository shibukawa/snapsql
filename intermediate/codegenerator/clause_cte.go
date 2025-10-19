package codegenerator

import (
	"fmt"

	"github.com/shibukawa/snapsql/parser"
)

// generateCTEClause processes WITH clause and generates instruction for CTE output.
// CTE is output as-is using RawTokens, just like other clauses.
// Directives and expressions within CTE are handled by recursion through RawTokens.
//
// Parameters:
//   - withClause: *parser.WithClause (CTE information)
//   - builder: *InstructionBuilder
//
// Returns:
//   - error: エラー
func generateCTEClause(
	withClause *parser.WithClause,
	builder *InstructionBuilder,
) error {
	if withClause == nil {
		return nil
	}

	// Get RawTokens from WithClause (which includes heading and body tokens)
	rawTokens := withClause.RawTokens()
	if len(rawTokens) == 0 {
		return nil
	}

	// Process all tokens (includes directives if present)
	if err := builder.ProcessTokens(rawTokens); err != nil {
		return fmt.Errorf("failed to process CTE tokens: %w", err)
	}

	return nil
}
