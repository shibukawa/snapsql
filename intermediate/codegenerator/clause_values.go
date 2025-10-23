package codegenerator

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/shibukawa/snapsql"
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

		if token.Directive != nil {
			switch token.Directive.Type {
			case "for":
				loopVar, collection, err := parseForDirectiveCondition(token.Directive.Condition)
				if err != nil {
					return fmt.Errorf("failed to parse for directive at %s: %w", token.Position.String(), err)
				}

				builder.AddForLoopStart(loopVar, collection, token.Position.String())
				i++
				continue

			case "end":
				if len(builder.loopStack) > 0 {
					loopEndPos := token.Position.String()
					for j := len(builder.instructions) - 1; j >= 0; j-- {
						instr := builder.instructions[j]
						if instr.Op == OpEmitUnlessBoundary {
							continue
						}

						if instr.Op == OpEmitStatic {
							trimmedVal := strings.TrimSpace(instr.Value)
							if trimmedVal == "" {
								continue
							}
							if trimmedVal == "," || trimmedVal == "AND" || trimmedVal == "OR" {
								continue
							}
						}

						if instr.Pos != "" {
							loopEndPos = instr.Pos
							break
						}
					}

					builder.AddForLoopEnd(loopEndPos)
					i++
					continue
				}
			}
		}

		// 変数展開ディレクティブ (/*= varname */) をチェック
		if token.Directive != nil && token.Directive.Type == "variable" {
			varName := token.Directive.Condition
			trimmedVar := strings.TrimSpace(varName)

			if insertStmt, ok := builder.context.Statement.(*parser.InsertIntoStatement); ok {
				descriptor := builder.lookupTypeDescriptor(token)
				if handledIdx, handled, err := handleAutoValuesDirective(builder, tokens, i, token, trimmedVar, descriptor, insertStmt, fields); err != nil {
					return err
				} else if handled {
					i = handledIdx
					continue
				}
			}

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

			if varInfo != nil {
				// 実際の値を評価して型判定 (リフレクションベース)
				isArrayType := false
				if varInfo.Value != nil {
					_, isArrayType = varInfo.Value.([]interface{})
				}

				if isArrayType {
					// 配列型：ループを生成
					// ループ要素名を生成 ("rows" -> "r")
					elementVar := strings.ToLower(string(varName[0]))

					// FOR ループ開始
					builder.AddForLoopStart(elementVar, varName, token.Position.String())

					// 配列型の場合、ディレクティブ自体の EMIT_EVAL は生成しない
					// （オブジェクトフィールド展開で各フィールドを個別に処理）

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
								Pos:   token.Position.String(),
							})

							// INTO句のカラムリストを取得
							insertStmt, ok := builder.context.Statement.(*parser.InsertIntoStatement)
							if ok && insertStmt != nil && insertStmt.Into != nil && len(insertStmt.Columns) > 0 {
								// オブジェクトフィールド展開：各カラムに対して loop_var.column を生成
								for colIdx, col := range insertStmt.Columns {
									if colIdx > 0 {
										// カンマを追加
										builder.AddInstruction(Instruction{
											Op:    OpEmitStatic,
											Value: ", ",
											Pos:   token.Position.String(),
										})
									}

									// loop_var.column_name の形式で EMIT_EVAL を生成
									fieldExpr := elementVar + "." + col.Name
									if _, err := builder.RegisterEmitEval(fieldExpr, token.Position.String()); err != nil {
										return err
									}
								}
							} else if len(fields) > 0 {
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
								Pos:   token.Position.String(),
							})
						}

						i = parenEnd
					}

					// OpEmitUnlessBoundary: ループ反復間のカンマを生成（ただし終了時は除外）
					builder.AddInstruction(Instruction{
						Op:    OpEmitUnlessBoundary,
						Value: ", ",
						Pos:   token.Position.String(),
					})

					// FOR ループ終了
					builder.AddForLoopEnd(token.Position.String())
				} else if varInfo != nil {
					// 非配列オブジェクト型をチェック
					isObjectType := false

					if varInfo.Value != nil {
						_, isMap := varInfo.Value.(map[string]interface{})
						_, isArray := varInfo.Value.([]interface{})
						// Object type if it's a map but NOT an array
						isObjectType = isMap && !isArray
					}

					if isObjectType {
						// 非配列オブジェクト型：ディレクティブ自体ではなくオブジェクトフィールドを展開
						// ループなし、直接フィールド展開
						i++ // ディレクティブトークンをスキップ

						// DUMMY_STARTから DUMMY_ENDまでをスキップ
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
								innerTokens := parenGroup[1 : len(parenGroup)-1]

								// 開き括弧を手動で追加
								builder.AddInstruction(Instruction{
									Op:    OpEmitStatic,
									Value: "(",
									Pos:   token.Position.String(),
								})

								// INTO句のカラムリストを取得
								insertStmt, ok := builder.context.Statement.(*parser.InsertIntoStatement)
								if ok && insertStmt != nil && insertStmt.Into != nil && len(insertStmt.Columns) > 0 {
									// オブジェクトフィールド展開：各カラムに対して varName.column を生成
									for colIdx, col := range insertStmt.Columns {
										if colIdx > 0 {
											// カンマを追加
											builder.AddInstruction(Instruction{
												Op:    OpEmitStatic,
												Value: ", ",
												Pos:   token.Position.String(),
											})
										}

										// varName.column_name の形式で EMIT_EVAL を生成
										fieldExpr := varName + "." + col.Name
										if _, err := builder.RegisterEmitEval(fieldExpr, token.Position.String()); err != nil {
											return err
										}
									}
								} else if len(fields) > 0 {
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
									Pos:   token.Position.String(),
								})
							}

							i = parenEnd
						}
					}
				}
				// スカラー型：通常処理
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
			builder.addStatic(token.Value, &token.Position)

			i++
		}
	}

	return nil
}

func handleAutoValuesDirective(builder *InstructionBuilder, tokens []tokenizer.Token, startIdx int, token tokenizer.Token, varName string, descriptor any, insertStmt *parser.InsertIntoStatement, fields []snapsql.SystemField) (int, bool, error) {
	if descriptor == nil || insertStmt == nil || insertStmt.Into == nil || len(insertStmt.Columns) == 0 || varName == "" {
		return startIdx, false, nil
	}

	resultType := DetermineEvalResultType(descriptor)
	switch resultType {
	case EvalResultTypeArray, EvalResultTypeArrayOfObject:
		return handleArrayDirective(builder, tokens, startIdx, token, varName, descriptor, insertStmt, fields)
	case EvalResultTypeObject:
		return handleObjectDirective(builder, tokens, startIdx, token, varName, descriptor, insertStmt, fields)
	default:
		return startIdx, false, nil
	}
}

func handleArrayDirective(builder *InstructionBuilder, tokens []tokenizer.Token, startIdx int, token tokenizer.Token, varName string, descriptor any, insertStmt *parser.InsertIntoStatement, fields []snapsql.SystemField) (int, bool, error) {
	nextIdx := skipDummyTokens(tokens, startIdx+1)
	if nextIdx >= len(tokens) || tokens[nextIdx].Value != "(" {
		return startIdx, false, nil
	}

	parenGroup, parenEnd := extractParenthesizedGroup(tokens, nextIdx)
	if len(parenGroup) <= 1 {
		return startIdx, false, nil
	}

	elementDescriptor := ExtractElementDescriptor(descriptor)

	loopVar := deriveLoopVariableName(varName)
	if loopVar == "" {
		loopVar = "__item"
	}

	objDesc, isObject := ExtractObjectDescriptor(elementDescriptor)

	var (
		mappings []columnMapping
		ok       bool
	)

	if isObject {
		mappings, ok = buildColumnMappings(loopVar, insertStmt.Columns, objDesc)
		if !ok {
			return startIdx, false, nil
		}
	} else {
		// Scalar arrays require exactly one column
		if len(insertStmt.Columns) != 1 {
			return startIdx, false, nil
		}
	}

	// Ready to emit instructions
	builder.AddForLoopStart(loopVar, varName, token.Position.String())

	builder.AddInstruction(Instruction{
		Op:    OpEmitStatic,
		Value: "(",
		Pos:   token.Position.String(),
	})

	if isObject {
		if err := emitColumnMappings(builder, token, mappings); err != nil {
			return startIdx, false, err
		}
	} else {
		if _, err := builder.RegisterEmitEvalWithDescriptor(token, loopVar, elementDescriptor); err != nil {
			return startIdx, false, err
		}
	}

	insertSystemFieldValues(builder, fields)

	builder.AddInstruction(Instruction{
		Op:    OpEmitStatic,
		Value: ")",
		Pos:   token.Position.String(),
	})

	builder.AddInstruction(Instruction{
		Op:    OpEmitUnlessBoundary,
		Value: ", ",
		Pos:   token.Position.String(),
	})

	builder.AddForLoopEnd(token.Position.String())

	return parenEnd, true, nil
}

func handleObjectDirective(builder *InstructionBuilder, tokens []tokenizer.Token, startIdx int, token tokenizer.Token, varName string, descriptor any, insertStmt *parser.InsertIntoStatement, fields []snapsql.SystemField) (int, bool, error) {
	nextIdx := skipDummyTokens(tokens, startIdx+1)
	if nextIdx >= len(tokens) || tokens[nextIdx].Value != "(" {
		return startIdx, false, nil
	}

	parenGroup, parenEnd := extractParenthesizedGroup(tokens, nextIdx)
	if len(parenGroup) <= 1 {
		return startIdx, false, nil
	}

	objDesc, ok := ExtractObjectDescriptor(descriptor)
	if !ok {
		return startIdx, false, nil
	}

	mappings, ok := buildColumnMappings(varName, insertStmt.Columns, objDesc)
	if !ok {
		return startIdx, false, nil
	}

	builder.AddInstruction(Instruction{
		Op:    OpEmitStatic,
		Value: "(",
		Pos:   token.Position.String(),
	})

	if err := emitColumnMappings(builder, token, mappings); err != nil {
		return startIdx, false, err
	}

	insertSystemFieldValues(builder, fields)

	builder.AddInstruction(Instruction{
		Op:    OpEmitStatic,
		Value: ")",
		Pos:   token.Position.String(),
	})

	return parenEnd, true, nil
}

func skipDummyTokens(tokens []tokenizer.Token, idx int) int {
	if idx < len(tokens) && tokens[idx].Type == tokenizer.DUMMY_START {
		idx++
		for idx < len(tokens) && tokens[idx].Type != tokenizer.DUMMY_END {
			idx++
		}

		if idx < len(tokens) {
			idx++
		}
	}

	return idx
}

func deriveLoopVariableName(varName string) string {
	if varName == "" {
		return ""
	}

	runes := []rune(varName)

	return strings.ToLower(string(runes[0]))
}

type columnMapping struct {
	Expression string
	Descriptor any
}

func buildColumnMappings(prefix string, columns []parser.FieldName, objectDesc map[string]any) ([]columnMapping, bool) {
	mappings := make([]columnMapping, 0, len(columns))
	for _, col := range columns {
		fieldName, fieldDesc, ok := matchObjectKey(col.Name, objectDesc)
		if !ok {
			return nil, false
		}

		expr := prefix + "." + fieldName
		mappings = append(mappings, columnMapping{Expression: expr, Descriptor: fieldDesc})
	}

	return mappings, true
}

func emitColumnMappings(builder *InstructionBuilder, token tokenizer.Token, mappings []columnMapping) error {
	for idx, mapping := range mappings {
		if idx > 0 {
			builder.AddInstruction(Instruction{
				Op:    OpEmitStatic,
				Value: ", ",
				Pos:   token.Position.String(),
			})
		}

		if _, err := builder.RegisterEmitEvalWithDescriptor(token, mapping.Expression, mapping.Descriptor); err != nil {
			return err
		}
	}

	return nil
}

func matchObjectKey(columnName string, objectDesc map[string]any) (string, any, bool) {
	candidates := fieldCandidates(columnName)
	for _, candidate := range candidates {
		if desc, ok := objectDesc[candidate]; ok {
			return candidate, desc, true
		}
	}

	return "", nil, false
}

func fieldCandidates(columnName string) []string {
	trimmed := strings.Trim(columnName, "`\"")
	if dot := strings.LastIndex(trimmed, "."); dot >= 0 && dot < len(trimmed)-1 {
		trimmed = trimmed[dot+1:]
	}

	base := trimmed
	lower := strings.ToLower(base)

	result := []string{base, lower}
	if strings.Contains(lower, "_") {
		lc := toLowerCamel(lower)
		uc := toUpperCamel(lower)
		result = append(result, lc, uc)
	}

	return uniqueStrings(result)
}

func toLowerCamel(input string) string {
	parts := strings.Split(input, "_")
	for i, part := range parts {
		parts[i] = strings.ToLower(part)
		if i > 0 && len(parts[i]) > 0 {
			runes := []rune(parts[i])
			runes[0] = unicode.ToUpper(runes[0])
			parts[i] = string(runes)
		}
	}

	return strings.Join(parts, "")
}

func toUpperCamel(input string) string {
	result := toLowerCamel(input)
	if result == "" {
		return result
	}

	runes := []rune(result)
	runes[0] = unicode.ToUpper(runes[0])

	return string(runes)
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))

	out := make([]string, 0, len(values))
	for _, v := range values {
		if v == "" {
			continue
		}

		if _, ok := seen[v]; ok {
			continue
		}

		seen[v] = struct{}{}
		out = append(out, v)
	}

	return out
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

func parseForDirectiveCondition(condition string) (string, string, error) {
	parts := strings.SplitN(condition, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("%w: invalid format (expected 'variable : expression')", ErrLoopMismatch)
	}

	loopVar := strings.TrimSpace(parts[0])
	collection := strings.TrimSpace(parts[1])

	if loopVar == "" || collection == "" {
		return "", "", fmt.Errorf("%w: empty variable or expression", ErrLoopMismatch)
	}

	return loopVar, collection, nil
}
