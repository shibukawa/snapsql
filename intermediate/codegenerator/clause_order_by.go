package codegenerator

import (
	"fmt"

	"github.com/shibukawa/snapsql/parser"
)

// generateOrderByClause は ORDER BY 句から命令列を生成する
func generateOrderByClause(clause *parser.OrderByClause, builder *InstructionBuilder) error {
	if clause == nil {
		return fmt.Errorf("%w: ORDER BY clause is nil", ErrClauseNil)
	}

	tokens := clause.RawTokens()
	// 例: ASC/DESC の処理、NULLS FIRST/LAST の処理など

	if err := builder.ProcessTokens(tokens); err != nil {
		return fmt.Errorf("failed to process tokens in ORDER BY clause: %w", err)
	}

	return nil
}
