package codegenerator

import (
	"fmt"

	"github.com/shibukawa/snapsql/parser"
)

// generateInsertIntoClause は INSERT INTO 句から命令列を生成する
//
// Parameters:
//   - into: *parser.InsertIntoClause - INSERT INTO節のAST
//   - columns: []parser.FieldName - カラム名リスト
//   - builder: *InstructionBuilder - 命令ビルダー
//
// Returns:
//   - error: エラー
//
// 生成例:
//
//	INPUT:  INSERT INTO users (id, name, email)
//	OUTPUT: EMIT_STATIC "INSERT INTO users (id, name, email)"
//
// 備考:
//   - INTO節のRawTokensをそのまま処理することで、テーブル名やエイリアスを自動処理
//   - ディレクティブが含まれる可能性に対応（理論上、テーブル名の動的生成は可能だが稀）
func generateInsertIntoClause(into *parser.InsertIntoClause, columns []parser.FieldName, builder *InstructionBuilder) error {
	if into == nil {
		return fmt.Errorf("%w: INSERT INTO clause is required", ErrMissingClause)
	}

	// RawTokens をそのまま処理
	tokens := into.RawTokens()
	if err := builder.ProcessTokens(tokens); err != nil {
		return fmt.Errorf("code generation: %w", err)
	}

	return nil
}
