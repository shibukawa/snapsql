package codegenerator

import (
	"fmt"

	"github.com/shibukawa/snapsql/parser"
)

// generateHavingClause は HAVING 句から命令列を生成する
func generateHavingClause(clause *parser.HavingClause, builder *InstructionBuilder) error {
	if clause == nil {
		return fmt.Errorf("%w: HAVING clause is nil", ErrClauseNil)
	}

	tokens := clause.RawTokens()

	// Phase 1: トークンをそのまま処理
	// 将来的には、ここでトークンのカスタマイズを行う
	// 例: 集約関数の処理、条件式の最適化など

	if err := builder.ProcessTokens(tokens); err != nil {
		return fmt.Errorf("failed to process tokens in HAVING clause: %w", err)
	}

	return nil
}
