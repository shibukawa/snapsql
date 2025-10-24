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
//   - システムフィールドはこの関数呼び出し後に別途追加される
func generateInsertIntoClause(into *parser.InsertIntoClause, columns []parser.FieldName, builder *InstructionBuilder, skipLeadingTrivia bool) error {
	if into == nil {
		return fmt.Errorf("%w: INSERT INTO clause is required", ErrMissingClause)
	}

	// RawTokens をそのまま処理
	tokens := into.RawTokens()

	// 既存のカラムリストからマップを作成（重複排除用）
	existingColumns := make(map[string]bool)
	for _, col := range columns {
		existingColumns[col.Name] = true
	}

	// システムフィールドをカラムリストに追加する場合
	fields := getInsertSystemFieldsFiltered(builder.context, existingColumns)

	if len(fields) > 0 {
		// 括弧の位置を見つける
		closingParenIdx := findClosingParenIndex(tokens)
		if closingParenIdx >= 0 {
			// 括弧の直前までのトークンを処理
			tokensBeforeParen := tokens[:closingParenIdx]
			options := []ProcessTokensOption{}
			if skipLeadingTrivia {
				options = append(options, WithSkipLeadingTrivia())
			}
			if err := builder.ProcessTokens(tokensBeforeParen, options...); err != nil {
				return fmt.Errorf("code generation: %w", err)
			}

			// システムフィールド名を追加
			insertSystemFieldNames(builder, fields)

			// 括弧を含むトークンを処理
			tokensFromParen := tokens[closingParenIdx:]
			if err := builder.ProcessTokens(tokensFromParen); err != nil {
				return fmt.Errorf("code generation: %w", err)
			}
		} else {
			// 括弧が見つからない場合は通常処理
			options := []ProcessTokensOption{}
			if skipLeadingTrivia {
				options = append(options, WithSkipLeadingTrivia())
			}
			if err := builder.ProcessTokens(tokens, options...); err != nil {
				return fmt.Errorf("code generation: %w", err)
			}
		}
	} else {
		// システムフィールドがない場合は通常処理
		options := []ProcessTokensOption{}
		if skipLeadingTrivia {
			options = append(options, WithSkipLeadingTrivia())
		}
		if err := builder.ProcessTokens(tokens, options...); err != nil {
			return fmt.Errorf("code generation: %w", err)
		}
	}

	return nil
}
