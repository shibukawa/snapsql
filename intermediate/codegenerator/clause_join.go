package codegenerator

import (
	"strings"

	"github.com/shibukawa/snapsql/tokenizer"
)

// generateJoinClause はプレースホルダー関数
// Phase 1 では、FROM句の RawTokens にすべての JOIN 情報が含まれているため、
// 別途の JOIN 処理は不要である
//
// 実装機能:
// - JOIN型の正規化 (LEFT OUTER JOIN → LEFT JOIN など)
// - ON条件内の方言変換
// - テーブル参照の管理と最適化

// normalizeJoinType はJOIN型を正規化する
// LEFT OUTER JOIN → LEFT JOIN
// RIGHT OUTER JOIN → RIGHT JOIN
// など
func normalizeJoinType(tokens []tokenizer.Token) []tokenizer.Token {
	result := make([]tokenizer.Token, 0, len(tokens))

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		keyword := strings.ToUpper(token.Value)

		// OUTER キーワードが JOIN の直後に来ている場合は削除
		// LEFT OUTER JOIN → LEFT JOIN
		// RIGHT OUTER JOIN → RIGHT JOIN
		if keyword == "OUTER" && len(result) > 0 {
			// 前のトークン（空白を除く）が LEFT または RIGHT か確認
			prevIdx := len(result) - 1
			for prevIdx >= 0 && isWhitespaceOrCommentToken(result[prevIdx]) {
				prevIdx--
			}

			if prevIdx >= 0 {
				prevKeyword := strings.ToUpper(result[prevIdx].Value)
				if prevKeyword == "LEFT" || prevKeyword == "RIGHT" {
					// OUTER を スキップ（result に追加しない）
					// 次の空白文字もスキップする場合がある
					if i+1 < len(tokens) && isWhitespaceOrCommentToken(tokens[i+1]) {
						i++ // 空白もスキップ
					}

					continue
				}
			}
		}

		result = append(result, token)
	}

	return result
}

// isWhitespaceOrCommentToken はトークンが空白またはコメントかを判定
func isWhitespaceOrCommentToken(token tokenizer.Token) bool {
	return token.Type == tokenizer.WHITESPACE ||
		token.Type == tokenizer.BLOCK_COMMENT ||
		token.Type == tokenizer.LINE_COMMENT
}
