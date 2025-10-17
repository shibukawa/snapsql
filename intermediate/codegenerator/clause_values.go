package codegenerator

import (
	"fmt"

	"github.com/shibukawa/snapsql/parser"
)

// generateValuesClause は VALUES 句から命令列を生成する
//
// Parameters:
//   - values: *parser.ValuesClause - VALUES節のAST
//   - builder: *InstructionBuilder - 命令ビルダー
//
// Returns:
//   - error: エラー
//
// 生成例:
//   INPUT:  VALUES (1, 'John', 'john@example.com'), (2, 'Jane', 'jane@example.com')
//   OUTPUT: EMIT_STATIC " VALUES "
//           EMIT_STATIC "(1, 'John', 'john@example.com')"
//           EMIT_UNLESS_BOUNDARY ", "
//           EMIT_STATIC "(2, 'Jane', 'jane@example.com')"
//
// ディレクティブ例:
//   INPUT:  VALUES (/*= user_id */ 1, /*= user_name */ 'John')
//   OUTPUT: EMIT_STATIC " VALUES ("
//           EMIT_EVAL (expr_index: 0)  # user_id
//           EMIT_STATIC ", "
//           EMIT_EVAL (expr_index: 1)  # user_name
//           EMIT_STATIC ")"
//
// 複数行とループ:
//   VALUES行の複数行対応やループディレクティブ（/*# for item in items */）
//   に対応している。InstructionBuilderのProcessTokensが自動的に処理する。
func generateValuesClause(values *parser.ValuesClause, builder *InstructionBuilder) error {
	if values == nil {
		return fmt.Errorf("%w: VALUES clause is required for VALUES-based INSERT", ErrMissingClause)
	}

	// RawTokens をそのまま処理
	// ディレクティブ、ループ、複数行の処理は ProcessTokens 内で実現される
	tokens := values.RawTokens()
	if err := builder.ProcessTokens(tokens); err != nil {
		return fmt.Errorf("code generation: %w", err)
	}

	return nil
}
