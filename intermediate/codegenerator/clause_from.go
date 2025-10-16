package codegenerator

import (
	"fmt"

	"github.com/shibukawa/snapsql/parser"
)

// generateFromClause は FROM 句から命令列を生成する
// Phase 1: 基本的なテーブルと FROM 句内サブクエリーをサポート
// Phase 2: WHERE/SELECT 句内のサブクエリーをサポート
// Phase 3: CTE のサポート
func generateFromClause(clause *parser.FromClause, builder *InstructionBuilder) error {
	if clause == nil {
		return fmt.Errorf("%w: FROM clause is nil", ErrClauseNil)
	}

	// RawTokens をそのまま処理（従来の動作を保つ）
	// サブクエリーの特殊処理は ProcessTokens 内で実現される
	tokens := clause.RawTokens()
	if err := builder.ProcessTokens(tokens); err != nil {
		return fmt.Errorf("%w: %v", ErrCodeGeneration, err)
	}

	return nil
}
