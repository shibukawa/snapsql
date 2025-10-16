package codegenerator

import (
	"fmt"

	"github.com/shibukawa/snapsql/parser"
)

// generateFromClause は FROM 句から命令列を生成する
func generateFromClause(clause *parser.FromClause, builder *InstructionBuilder) error {
	if clause == nil {
		return fmt.Errorf("%w: FROM clause is nil", ErrClauseNil)
	}

	tokens := clause.RawTokens()

	// Phase 1: トークンをそのまま処理
	// RawTokens にはすべての JOIN 情報が含まれているため、
	// 別途の JOIN 処理は不要（将来的には最適化や方言変換を追加）

	if err := builder.ProcessTokens(tokens); err != nil {
		return fmt.Errorf("failed to process tokens in FROM clause: %w", err)
	}

	return nil
}
