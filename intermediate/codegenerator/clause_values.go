package codegenerator

import (
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/tokenizer"
)

// generateValuesClause は VALUES 句から命令列を生成する
//
// 処理フロー：
// 1. VALUES キーワードを emit
// 2. トークンを逐次処理し、各要素（括弧、ディレクティブ、値）を判定
// 3. 配列型の /*= varname */ ディレクティブの場合、ループを展開
// 4. 各値グループ後にシステムフィールド値を挿入
//
// Parameters:
//   - values: *parser.ValuesClause - VALUES節のAST
//   - builder: *InstructionBuilder - 命令ビルダー
//   - columnNames: []parser.FieldName - カラム名リスト（重複排除用）
//
// Returns:
//   - error: エラー
func generateValuesClause(values *parser.ValuesClause, builder *InstructionBuilder, columnNames []parser.FieldName) error {
	if values == nil {
		return fmt.Errorf("%w: VALUES clause is required for VALUES-based INSERT", ErrMissingClause)
	}

	tokens := values.RawTokens()

	// 既存のカラムリストからマップを作成（重複排除用）
	existingColumns := make(map[string]bool)
	for _, col := range columnNames {
		existingColumns[col.Name] = true
	}

	// システムフィールド値が必要かチェック
	fields := getInsertSystemFieldsFiltered(builder.context, existingColumns)

	// Phase 1: VALUES キーワードの送信
	// VALUES句の開始まで（最初の括弧や変数展開ディレクティブまで）のトークンを処理
	i := 0
	var preambleTokens []tokenizer.Token
	for i < len(tokens) {
		token := tokens[i]

		// ディレクティブ（括弧、ループなど）に到達したら終了
		if isStartOfValueGroup(token) {
			break
		}

		// 通常のトークン（VALUES キーワードなど）を収集
		preambleTokens = append(preambleTokens, token)
		i++
	}

	// 収集したトークンを一括処理
	if len(preambleTokens) > 0 {
		if err := builder.ProcessTokens(preambleTokens); err != nil {
			return fmt.Errorf("code generation: %w", err)
		}
	}

	// Phase 2: 値グループを逐次処理
	// 各値グループの処理：
	// - 括弧で囲まれた値 (val1, val2, ...)
	// - または変数展開 /*= rows */ で複数行を展開
	for i < len(tokens) {
		token := tokens[i]

		// 変数展開ディレクティブ (/*= varname */) をチェック
		if token.Directive != nil && token.Directive.Type == "variable" {
			varName := token.Directive.Condition
			envIndex := builder.getCurrentEnvironmentIndex()

			env := builder.context.CELEnvironments[envIndex]

			// 変数情報を取得
			var varInfo *CELVariableInfo

			for j := range env.AdditionalVariables {
				if env.AdditionalVariables[j].Name == varName {
					varInfo = &env.AdditionalVariables[j]
					break
				}
			}

			if varInfo != nil && isArrayType(varInfo.Type) {
				// 配列型：ループを生成
				fmt.Printf("DEBUG: generateValuesClause: variable='%s', type='%s' is array, generating loop\n", varName, varInfo.Type)

				// ループ要素名を生成 ("rows" -> "r")
				elementVar := strings.ToLower(string(varName[0]))

				// FOR ループ開始
				builder.AddForLoopStart(elementVar, varName, token.Position.String())

				// 変数ディレクティブトークンを処理して EMIT_EVAL を生成
				if token.Directive != nil {
					if _, err := builder.RegisterEmitEval(token.Directive.Condition, token.Position.String()); err != nil {
						return err
					}
				}

				// 次のトークンを処理：括弧内の値
				i++

				// DUMMY_STARTから DUMMY_ENDまでをスキップ
				// パーサーがディレクティブの後にDUMMY_START, <dummy_value>, DUMMY_ENDを挿入
				if i < len(tokens) && tokens[i].Type == tokenizer.DUMMY_START {
					i++ // Skip DUMMY_START
					// Skip all tokens until DUMMY_END
					for i < len(tokens) && tokens[i].Type != tokenizer.DUMMY_END {
						i++
					}
					// Skip DUMMY_END
					if i < len(tokens) && tokens[i].Type == tokenizer.DUMMY_END {
						i++
					}
				}

				if i < len(tokens) && tokens[i].Value == "(" {
					// 括弧内の値をまとめて処理
					parenGroup, parenEnd := extractParenthesizedGroup(tokens, i)
					if len(parenGroup) > 1 {
						// 括弧をスキップして、括弧内の全トークンを抽出
						// parenGroup[0] は開き括弧
						// parenGroup[len-1] は閉じ括弧
						innerTokens := parenGroup[1 : len(parenGroup)-1]

						// 開き括弧を手動で追加
						builder.AddInstruction(Instruction{
							Op:    OpEmitStatic,
							Value: "(",
							Pos:   tokens[i].Position.String(),
						})

						if len(fields) > 0 {
							// システムフィールド挿入位置を探す
							lastParenIdx := findLastClosingParenInGroup(innerTokens)
							if lastParenIdx >= 0 {
								// 括弧の前までを処理
								if err := builder.ProcessTokens(innerTokens[:lastParenIdx]); err != nil {
									return err
								}
								insertSystemFieldValues(builder, fields)
								// 括弧から後ろを処理
								if err := builder.ProcessTokens(innerTokens[lastParenIdx:]); err != nil {
									return err
								}
							} else {
								// すべての内部トークンを処理
								if err := builder.ProcessTokens(innerTokens); err != nil {
									return err
								}
								// 末尾にシステムフィールド値を挿入
								insertSystemFieldValues(builder, fields)
							}
						} else {
							// すべての内部トークンを処理
							if err := builder.ProcessTokens(innerTokens); err != nil {
								return err
							}
						}

						// 閉じ括弧を手動で追加
						builder.AddInstruction(Instruction{
							Op:    OpEmitStatic,
							Value: ")",
							Pos:   tokens[parenEnd-1].Position.String(),
						})
					}

					i = parenEnd
				}

				// FOR ループ終了
				builder.AddForLoopEnd(token.Position.String())
			} else {
				// スカラー型：通常処理
				fmt.Printf("DEBUG: generateValuesClause: variable='%s' is scalar, no loop\n", varName)

				// 変数ディレクティブトークンを処理して EMIT_EVAL を生成
				if token.Directive != nil {
					if _, err := builder.RegisterEmitEval(token.Directive.Condition, token.Position.String()); err != nil {
						return err
					}
				}

				i++

				// 次のトークンが括弧なら処理
				if i < len(tokens) && tokens[i].Value == "(" {
					parenGroup, parenEnd := extractParenthesizedGroup(tokens, i)
					if len(parenGroup) > 1 {
						// innerTokens: 括弧トークンをスキップして括弧内の値だけを抽出
						innerTokens := parenGroup[1 : len(parenGroup)-1]

						// 開き括弧を手動で追加
						builder.AddInstruction(Instruction{
							Op:    OpEmitStatic,
							Value: "(",
							Pos:   tokens[i].Position.String(),
						})

						if len(fields) > 0 {
							lastParenIdx := findLastClosingParenInGroup(innerTokens)
							if lastParenIdx >= 0 {
								if err := builder.ProcessTokens(innerTokens[:lastParenIdx]); err != nil {
									return err
								}
								insertSystemFieldValues(builder, fields)
								if err := builder.ProcessTokens(innerTokens[lastParenIdx:]); err != nil {
									return err
								}
							} else {
								if err := builder.ProcessTokens(innerTokens); err != nil {
									return err
								}
								insertSystemFieldValues(builder, fields)
							}
						} else {
							if err := builder.ProcessTokens(innerTokens); err != nil {
								return err
							}
						}

						// 閉じ括弧を手動で追加
						builder.AddInstruction(Instruction{
							Op:    OpEmitStatic,
							Value: ")",
							Pos:   tokens[parenEnd-1].Position.String(),
						})
					}

					i = parenEnd
				}
			}
		} else if token.Value == "(" {
			// 通常の括弧で囲まれた値グループ
			parenGroup, parenEnd := extractParenthesizedGroup(tokens, i)
			if len(parenGroup) > 1 {
				// innerTokens: 括弧トークンをスキップして括弧内の値だけを抽出
				innerTokens := parenGroup[1 : len(parenGroup)-1]

				// 開き括弧を手動で追加
				builder.AddInstruction(Instruction{
					Op:    OpEmitStatic,
					Value: "(",
					Pos:   tokens[i].Position.String(),
				})

				if len(fields) > 0 {
					lastParenIdx := findLastClosingParenInGroup(innerTokens)
					if lastParenIdx >= 0 {
						if err := builder.ProcessTokens(innerTokens[:lastParenIdx]); err != nil {
							return err
						}
						insertSystemFieldValues(builder, fields)
						if err := builder.ProcessTokens(innerTokens[lastParenIdx:]); err != nil {
							return err
						}
					} else {
						if err := builder.ProcessTokens(innerTokens); err != nil {
							return err
						}
						insertSystemFieldValues(builder, fields)
					}
				} else {
					if err := builder.ProcessTokens(innerTokens); err != nil {
						return err
					}
				}

				// 閉じ括弧を手動で追加
				builder.AddInstruction(Instruction{
					Op:    OpEmitStatic,
					Value: ")",
					Pos:   tokens[parenEnd-1].Position.String(),
				})
			}

			i = parenEnd
		} else {
			// その他のトークン（カンマなど）
			// その他のトークン（カンマなど）は静的に出力
			builder.addStatic(token.Value, token.Position)

			i++
		}
	}

	return nil
}

// isStartOfValueGroup checks if a token marks the start of a value group
func isStartOfValueGroup(token tokenizer.Token) bool {
	// 括弧か変数展開ディレクティブで始まる
	if token.Value == "(" {
		return true
	}

	if token.Directive != nil && token.Directive.Type == "variable" {
		return true
	}

	return false
}

// extractParenthesizedGroup extracts tokens from opening '(' to closing ')', inclusive
// Returns the group and the index after the closing paren
func extractParenthesizedGroup(tokens []tokenizer.Token, startIdx int) ([]tokenizer.Token, int) {
	if startIdx >= len(tokens) || tokens[startIdx].Value != "(" {
		return nil, startIdx + 1
	}

	group := make([]tokenizer.Token, 0)
	depth := 0

	for i := startIdx; i < len(tokens); i++ {
		token := tokens[i]
		group = append(group, token)

		switch token.Value {
		case "(":
			depth++
		case ")":
			depth--
			if depth == 0 {
				// 括弧が閉じた
				return group, i + 1
			}
		}
	}

	return group, len(tokens)
}

// findLastClosingParenInGroup finds the index of the last ')' token in a token group
func findLastClosingParenInGroup(tokens []tokenizer.Token) int {
	for i := len(tokens) - 1; i >= 0; i-- {
		if tokens[i].Value == ")" {
			return i
		}
	}

	return -1
}

// isArrayType checks if a type string represents an array
func isArrayType(typeStr string) bool {
	return strings.HasPrefix(typeStr, "[") || strings.HasSuffix(typeStr, "[]")
}
