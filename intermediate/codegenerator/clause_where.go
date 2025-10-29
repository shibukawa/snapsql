package codegenerator

import (
	"fmt"

	"github.com/shibukawa/snapsql/parser"
)

// generateWhereClause は WHERE 句から命令列を生成する
func generateWhereClause(clause *parser.WhereClause, builder *InstructionBuilder) error {
	if clause == nil {
		return fmt.Errorf("%w: WHERE clause is nil", ErrClauseNil)
	}

	tokens := clause.RawTokens()

	// Phase 1: トークンをそのまま処理
	// 将来的には、ここでトークンのカスタマイズを行う
	// 例: 条件式の最適化、サブクエリの処理など

	if err := builder.ProcessTokens(tokens); err != nil {
		return fmt.Errorf("failed to process tokens in WHERE clause: %w", err)
	}

	// WHERE句の終了時に BOUNDARY を追加
	// （ただし、末尾が END の場合は追加しない）
	builder.AddBoundary()

	return nil
}
