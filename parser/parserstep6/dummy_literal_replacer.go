package parserstep6

import (
	"fmt"
	"strconv"
	"strings"

	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

// replaceDummyLiterals はステートメント内のコメント形式の変数参照をリテラルに置き換えます
// 例: /*= value */ → /*= value */ DUMMY_START 42 DUMMY_END
// 例: /*$ value */ → /*$ value */ 42
func replaceDummyLiterals(statement cmn.StatementNode, paramNamespace *cmn.Namespace, perr *cmn.ParseError) {
	// nilチェック
	if statement == nil || paramNamespace == nil || perr == nil {
		return
	}

	// 各句を処理
	for _, clause := range statement.Clauses() {
		if clause != nil {
			processClauseTokens(clause, paramNamespace, perr)
		}
	}
}

// processClauseTokens は句内のトークンを処理し、変数参照をリテラルに置き換えます
func processClauseTokens(clause cmn.ClauseNode, paramNs *cmn.Namespace, perr *cmn.ParseError) {
	tokens := clause.RawTokens()
	
	// 変数参照を検出し、リテラルを挿入または置換するインデックスと値のリストを作成
	var insertions []tokenInsertion
	var replacements []tokenReplacement
	
	for i, token := range tokens {
		if token.Type == tokenizer.BLOCK_COMMENT {
			if strings.HasPrefix(token.Value, "/*=") && strings.HasSuffix(token.Value, "*/") {
				// /*= value */ 形式の変数参照（ダミーリテラル）
				varName := extractExpressionFromDirective(token.Value, "/*=", "*/")
				if varName == "" {
					perr.Add(fmt.Errorf("%w at %s: invalid variable directive format", cmn.ErrInvalidForSnapSQL, token.Position.String()))
					continue
				}
				
				// 変数の値を評価
				value, valueType, err := paramNs.Eval(varName)
				if err != nil {
					perr.Add(fmt.Errorf("undefined variable in expression '%s': %w at %s", varName, err, token.Position.String()))
					continue
				}
				
				// 値をリテラル表現に変換
				literalTokens := createLiteralTokens(value, valueType, token.Position)
				if len(literalTokens) == 0 {
					perr.Add(fmt.Errorf("failed to create literal tokens for variable '%s' with type '%s'", varName, valueType))
					continue
				}
				
				// 挿入位置と挿入するトークンを記録
				insertions = append(insertions, tokenInsertion{
					index:  i, // トークンの配列内のインデックス
					tokens: literalTokens,
				})
			} else if strings.HasPrefix(token.Value, "/*$") && strings.HasSuffix(token.Value, "*/") {
				// /*$ value */ 形式の変数参照（正式な値）
				varName := extractExpressionFromDirective(token.Value, "/*$", "*/")
				if varName == "" {
					perr.Add(fmt.Errorf("%w at %s: invalid const directive format", cmn.ErrInvalidForSnapSQL, token.Position.String()))
					continue
				}
				
				// 変数の値を評価
				value, valueType, err := paramNs.Eval(varName)
				if err != nil {
					perr.Add(fmt.Errorf("undefined variable in expression '%s': %w at %s", varName, err, token.Position.String()))
					continue
				}
				
				// 値をリテラル表現に変換（DUMMY_STARTとDUMMY_ENDなし）
				valueToken := createValueToken(value, valueType, token.Position)
				if valueToken.Type == tokenizer.EOF { // 無効なトークンタイプとしてEOFを使用
					perr.Add(fmt.Errorf("failed to create value token for variable '%s' with type '%s'", varName, valueType))
					continue
				}
				
				// 既存のダミーリテラルを検索して削除対象にする
				if i+1 < len(tokens) && tokens[i+1].Type == tokenizer.DUMMY_LITERAL {
					// ダミーリテラルを置換する
					replacements = append(replacements, tokenReplacement{
						startIndex: i + 1,
						endIndex:   i + 2, // ダミーリテラルは1つのトークン
						token:      valueToken,
					})
				} else {
					// コメントの後に値を挿入する
					insertions = append(insertions, tokenInsertion{
						index:  i,
						tokens: []tokenizer.Token{valueToken},
					})
				}
			}
		}
	}
	
	// 置換を実行（後ろから実行することで、インデックスのずれを防ぐ）
	for i := len(replacements) - 1; i >= 0; i-- {
		replacement := replacements[i]
		replaceTokens(clause, replacement.startIndex, replacement.endIndex, replacement.token)
	}
	
	// 挿入を実行（後ろから実行することで、インデックスのずれを防ぐ）
	for i := len(insertions) - 1; i >= 0; i-- {
		insertion := insertions[i]
		// ClauseNodeのInsertTokensAfterIndexメソッドを使用してトークンを挿入
		clause.InsertTokensAfterIndex(insertion.index, insertion.tokens)
	}
}

// tokenInsertion はトークンの挿入情報を表します
type tokenInsertion struct {
	index  int              // 挿入位置
	tokens []tokenizer.Token // 挿入するトークン
}

// tokenReplacement はトークンの置換情報を表します
type tokenReplacement struct {
	startIndex int             // 置換開始位置
	endIndex   int             // 置換終了位置
	token      tokenizer.Token // 置換するトークン
}

// replaceTokens は指定された範囲のトークンを新しいトークンに置き換えます
func replaceTokens(clause cmn.ClauseNode, startIndex, endIndex int, newToken tokenizer.Token) {
	// ClauseNodeのReplaceTokensメソッドを使用してトークンを置換
	clause.ReplaceTokens(startIndex, endIndex, newToken)
}

// createLiteralTokens は値と型に基づいてダミーリテラルトークンを作成します
func createLiteralTokens(value any, valueType string, pos tokenizer.Position) []tokenizer.Token {
	// DUMMY_STARTトークン
	startToken := tokenizer.Token{
		Type:     tokenizer.DUMMY_START,
		Value:    "DUMMY_START",
		Position: pos,
	}
	
	// DUMMY_ENDトークン
	endToken := tokenizer.Token{
		Type:     tokenizer.DUMMY_END,
		Value:    "DUMMY_END",
		Position: pos,
	}
	
	// 値のトークン
	valueToken := createValueToken(value, valueType, pos)
	
	return []tokenizer.Token{startToken, valueToken, endToken}
}

// createValueToken は値と型に基づいて値のトークンを作成します
func createValueToken(value any, valueType string, pos tokenizer.Position) tokenizer.Token {
	var valueToken tokenizer.Token
	
	switch valueType {
	case "int":
		// 整数リテラル
		valueToken = tokenizer.Token{
			Type:     tokenizer.NUMBER,
			Value:    fmt.Sprintf("%d", value),
			Position: pos,
		}
	case "float":
		// 浮動小数点リテラル
		valueToken = tokenizer.Token{
			Type:     tokenizer.NUMBER,
			Value:    strconv.FormatFloat(value.(float64), 'f', -1, 64),
			Position: pos,
		}
	case "string":
		// 文字列リテラル（シングルクォートで囲む）
		valueToken = tokenizer.Token{
			Type:     tokenizer.STRING,
			Value:    fmt.Sprintf("'%s'", escapeString(value.(string))),
			Position: pos,
		}
	case "bool":
		// 真偽値リテラル
		var boolStr string
		if value.(bool) {
			boolStr = "TRUE"
		} else {
			boolStr = "FALSE"
		}
		valueToken = tokenizer.Token{
			Type:     tokenizer.BOOLEAN,
			Value:    boolStr,
			Position: pos,
		}
	default:
		// その他の型は文字列として扱う
		valueToken = tokenizer.Token{
			Type:     tokenizer.STRING,
			Value:    fmt.Sprintf("'%s'", escapeString(fmt.Sprintf("%v", value))),
			Position: pos,
		}
	}
	
	return valueToken
}

// escapeString は文字列内のシングルクォートをエスケープします
func escapeString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
