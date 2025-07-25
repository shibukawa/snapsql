package parserstep4

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/parser/parserstep1"
	"github.com/shibukawa/snapsql/parser/parserstep2"
	"github.com/shibukawa/snapsql/parser/parserstep3"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

// TestUpdateWithDummyTokensFromSource tests UPDATE statements with dummy tokens
// by tokenizing text source code.
func TestUpdateWithDummyTokensFromSource(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		wantErr   bool
		wantTable cmn.TableReference
		wantSets  []string // フィールド名のリスト
	}{
		{
			name:      "update with dummy literal",
			sql:       "UPDATE users SET name = /*= name */ WHERE id = /*= id */",
			wantErr:   false,
			wantTable: cmn.TableReference{Name: "users"},
			wantSets:  []string{"name"},
		},
		{
			name:      "update with const literal",
			sql:       "UPDATE users SET name = /*$ name */ WHERE id = /*$ id */",
			wantErr:   false,
			wantTable: cmn.TableReference{Name: "users"},
			wantSets:  []string{"name"},
		},
		{
			name:      "update with multiple fields and dummy literals",
			sql:       "UPDATE users SET name = /*= name */, age = /*= age */ WHERE id = /*= id */",
			wantErr:   false,
			wantTable: cmn.TableReference{Name: "users"},
			wantSets:  []string{"name", "age"},
		},
		{
			name:      "update with mix of dummy and const literals",
			sql:       "UPDATE users SET name = /*= name */, age = /*$ age */ WHERE id = /*= id */",
			wantErr:   false,
			wantTable: cmn.TableReference{Name: "users"},
			wantSets:  []string{"name", "age"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テキストソースコードからトークン列を生成
			tokens, err := tok.Tokenize(tt.sql)
			assert.NoError(t, err)

			// デバッグ用：トークンを表示
			t.Log("Original Tokens:")
			for i, token := range tokens {
				t.Logf("[%d] %s: %s", i, token.Type, token.Value)
			}

			// parserstep1を実行
			processedTokens, err := parserstep1.Execute(tokens)
			assert.NoError(t, err)

			// デバッグ用：処理後のトークンを表示
			t.Log("Processed Tokens after parserstep1:")
			for i, token := range processedTokens {
				t.Logf("[%d] %s: %s", i, token.Type, token.Value)
			}

			// parserstep2を実行
			stmt, err := parserstep2.Execute(processedTokens)
			if err != nil {
				t.Fatalf("parserstep2.Execute failed: %v", err)
			}

			// parserstep3を実行
			err = parserstep3.Execute(stmt)
			if err != nil {
				t.Fatalf("parserstep3.Execute failed: %v", err)
			}

			// UpdateStatementにキャスト
			updateStmt, ok := stmt.(*cmn.UpdateStatement)
			if !ok {
				t.Fatalf("stmt is not *cmn.UpdateStatement")
			}

			// finalizeUpdateClauseを実行
			perr := &cmn.ParseError{}
			finalizeUpdateClause(updateStmt.Update, perr)

			// finalizeSetClauseを実行
			finalizeSetClause(updateStmt.Set, perr)

			// 結果を検証
			if tt.wantErr {
				assert.NotEqual(t, 0, len(perr.Errors))
			} else {
				for _, e := range perr.Errors {
					t.Logf("Error: %s", e.Error())
				}
				assert.Equal(t, 0, len(perr.Errors))
				assert.Equal(t, tt.wantTable, updateStmt.Update.Table)

				// SET句のフィールド名を検証
				fieldNames := make([]string, 0, len(updateStmt.Set.Assigns))
				for _, assign := range updateStmt.Set.Assigns {
					fieldNames = append(fieldNames, assign.FieldName)
				}
				assert.Equal(t, tt.wantSets, fieldNames)
			}
		})
	}
}
