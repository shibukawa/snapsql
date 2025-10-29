package codegenerator

import (
	"fmt"

	"github.com/shibukawa/snapsql/parser"
)

// generateOnConflictClause は ON CONFLICT 句から命令列を生成する
//
// Parameters:
//   - onConflict: *parser.OnConflictClause - ON CONFLICT節のAST
//   - builder: *InstructionBuilder - 命令ビルダー
//
// Returns:
//   - error: エラー
//
// 生成例:
//
//	INPUT:  ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name
//	OUTPUT: EMIT_STATIC " ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name"
//
// 備考:
//   - PostgreSQL固有の機能（Phase 3実装）
//   - DO NOTHING と DO UPDATE SET ... の両方をサポート
//   - EXCLUDED 疑似テーブルの参照を適切に処理
//   - 制約名またはカラムリストでの競合検出をサポート
//   - ON CONFLICT節はオプショナルなので、nilの場合はエラーではなくスキップ
func generateOnConflictClause(onConflict *parser.OnConflictClause, builder *InstructionBuilder) error {
	if onConflict == nil {
		// ON CONFLICT節はオプショナルなので、nilの場合は何もしない
		return nil
	}

	// RawTokens をそのまま処理
	tokens := onConflict.RawTokens()
	if err := builder.ProcessTokens(tokens); err != nil {
		return fmt.Errorf("code generation: %w", err)
	}

	return nil
}
