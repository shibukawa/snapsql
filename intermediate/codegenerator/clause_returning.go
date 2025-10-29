package codegenerator

import (
	"fmt"

	"github.com/shibukawa/snapsql/parser"
)

// generateReturningClause は RETURNING 句から命令列を生成する
//
// Parameters:
//   - returning: *parser.ReturningClause - RETURNING節のAST
//   - builder: *InstructionBuilder - 命令ビルダー
//
// Returns:
//   - error: エラー
//
// 生成例:
//
//	INPUT:  RETURNING id, name, created_at
//	OUTPUT: EMIT_STATIC " RETURNING id, name, created_at"
//
// 備考:
//   - PostgreSQLとSQLiteでサポート
//   - MariaDB は INSERT に限定して RETURNING をサポート
//   - MySQL は RETURNING をサポートしない
//   - SELECT句と同様のカラムリスト処理
//   - ワイルドカード（*）のサポート
//   - 式や関数呼び出しのサポート（例: RETURNING id, UPPER(name)）
//   - RETURNING節はオプショナルなので、nilの場合はエラーではなくスキップ
func generateReturningClause(returning *parser.ReturningClause, builder *InstructionBuilder) error {
	if returning == nil {
		// RETURNING節はオプショナルなので、nilの場合は何もしない
		return nil
	}

	// RawTokens をそのまま処理
	tokens := returning.RawTokens()
	if err := builder.ProcessTokens(tokens); err != nil {
		return fmt.Errorf("code generation: %w", err)
	}

	return nil
}
