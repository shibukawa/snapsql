package codegenerator

import (
	"fmt"

	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/tokenizer"
)

// generateValuesClause は VALUES 句から命令列を生成する
//
// Parameters:
//   - values: *parser.ValuesClause - VALUES節のAST
//   - builder: *InstructionBuilder - 命令ビルダー
//   - columnNames: []parser.FieldName - カラム名リスト（重複排除用）
//
// Returns:
//   - error: エラー
//
// 生成例:
//
//	INPUT:  VALUES (1, 'John', 'john@example.com'), (2, 'Jane', 'jane@example.com')
//	OUTPUT: EMIT_STATIC " VALUES "
//	        EMIT_STATIC "(1, 'John', 'john@example.com')"
//	        EMIT_UNLESS_BOUNDARY ", "
//	        EMIT_STATIC "(2, 'Jane', 'jane@example.com')"
//
// ディレクティブ例:
//
//	INPUT:  VALUES (/*= user_id */ 1, /*= user_name */ 'John')
//	OUTPUT: EMIT_STATIC " VALUES ("
//	        EMIT_EVAL (expr_index: 0)  # user_id
//	        EMIT_STATIC ", "
//	        EMIT_EVAL (expr_index: 1)  # user_name
//	        EMIT_STATIC ")"
//
// 複数行とループ:
//
//	VALUES行の複数行対応やループディレクティブ（/*# for item in items */）
//	に対応している。InstructionBuilderのProcessTokensが自動的に処理する。
//
// システムフィールド:
//
//	システムフィールド（created_at等）の値は逐次送り出し式で処理される
func generateValuesClause(values *parser.ValuesClause, builder *InstructionBuilder, columnNames []parser.FieldName) error {
	if values == nil {
		return fmt.Errorf("%w: VALUES clause is required for VALUES-based INSERT", ErrMissingClause)
	}

	// RawTokens をそのまま処理
	tokens := values.RawTokens()

	// 既存のカラムリストからマップを作成（重複排除用）
	existingColumns := make(map[string]bool)
	for _, col := range columnNames {
		existingColumns[col.Name] = true
	}

	// システムフィールド値が必要かチェック
	fields := getInsertSystemFieldsFiltered(builder.context, existingColumns)

	if len(fields) > 0 {
		// 最後の行の閉じ括弧位置を見つける
		lastClosingParenIdx := findLastClosingParenIndex(tokens)
		if lastClosingParenIdx >= 0 {
			// 最後の閉じ括弧の直前までのトークンを処理
			tokensBeforeParen := tokens[:lastClosingParenIdx]
			if err := builder.ProcessTokens(tokensBeforeParen); err != nil {
				return fmt.Errorf("code generation: %w", err)
			}

			// システムフィールド値を追加
			insertSystemFieldValues(builder, fields)

			// 最後の閉じ括弧を含むトークンを処理
			tokensFromParen := tokens[lastClosingParenIdx:]
			if err := builder.ProcessTokens(tokensFromParen); err != nil {
				return fmt.Errorf("code generation: %w", err)
			}
		} else {
			// 括弧が見つからない場合は通常処理
			if err := builder.ProcessTokens(tokens); err != nil {
				return fmt.Errorf("code generation: %w", err)
			}
		}
	} else {
		// システムフィールドがない場合は通常処理
		if err := builder.ProcessTokens(tokens); err != nil {
			return fmt.Errorf("code generation: %w", err)
		}
	}

	return nil
}

// findLastClosingParenIndex finds the index of the last closing parenthesis in a token slice.
// Returns the index of the token containing the last ')', or -1 if not found.
func findLastClosingParenIndex(tokens []tokenizer.Token) int {
	for i := len(tokens) - 1; i >= 0; i-- {
		token := tokens[i]
		// Check if this token contains a closing paren
		for _, ch := range token.Value {
			if ch == ')' {
				return i
			}
		}
	}
	return -1
}
