package codegenerator

import (
	"fmt"

	"github.com/shibukawa/snapsql/parser"
)

// generateGroupByClause は GROUP BY 句から命令列を生成する
func generateGroupByClause(clause *parser.GroupByClause, builder *InstructionBuilder) error {
	if clause == nil {
		return fmt.Errorf("%w: GROUP BY clause is nil", ErrClauseNil)
	}

	tokens := clause.RawTokens()

	// Phase 1: トークンをそのまま処理
	// 将来的には、ここでトークンのカスタマイズを行う
	// 例: ROLLUP、CUBE、GROUPING SETS の処理など

	if err := builder.ProcessTokens(tokens); err != nil {
		return fmt.Errorf("failed to process tokens in GROUP BY clause: %w", err)
	}

	return nil
}
