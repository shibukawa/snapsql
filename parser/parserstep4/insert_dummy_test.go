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

// TestInsertWithDummyTokensFromSource tests INSERT statements with dummy tokens
// by tokenizing text source code.
func TestInsertWithDummyTokensFromSource(t *testing.T) {
	tests := []struct {
		name        string
		sql         string
		wantErr     bool
		wantTable   cmn.TableReference
		wantColumns []string
	}{
		{
			name:        "insert with dummy literal",
			sql:         "INSERT INTO users (id, name) VALUES (/*= id */, /*= name */)",
			wantErr:     false,
			wantTable:   cmn.TableReference{Name: "users"},
			wantColumns: []string{"id", "name"},
		},
		{
			name:        "insert with const literal",
			sql:         "INSERT INTO users (id, name) VALUES (/*$ id */, /*$ name */)",
			wantErr:     false,
			wantTable:   cmn.TableReference{Name: "users"},
			wantColumns: []string{"id", "name"},
		},
		{
			name:        "insert with multiple rows and dummy literals",
			sql:         "INSERT INTO users (id, name) VALUES (/*= id1 */, /*= name1 */), (/*= id2 */, /*= name2 */)",
			wantErr:     false,
			wantTable:   cmn.TableReference{Name: "users"},
			wantColumns: []string{"id", "name"},
		},
		{
			name:        "insert with mix of dummy and const literals",
			sql:         "INSERT INTO users (id, name) VALUES (/*= id */, /*$ name */)",
			wantErr:     false,
			wantTable:   cmn.TableReference{Name: "users"},
			wantColumns: []string{"id", "name"},
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

			// InsertIntoStatementにキャスト
			insertStmt, ok := stmt.(*cmn.InsertIntoStatement)
			if !ok {
				t.Fatalf("stmt is not *cmn.InsertIntoStatement")
			}

			// finalizeInsertIntoClauseを実行
			perr := &cmn.ParseError{}
			finalizeInsertIntoClause(insertStmt.Into, insertStmt.Select, perr)

			// 結果を検証
			if tt.wantErr {
				assert.NotEqual(t, 0, len(perr.Errors))
			} else {
				for _, e := range perr.Errors {
					t.Logf("Error: %s", e.Error())
				}
				assert.Equal(t, 0, len(perr.Errors))
				assert.Equal(t, tt.wantTable, insertStmt.Into.Table)
				assert.Equal(t, tt.wantColumns, insertStmt.Into.Columns)
			}
		})
	}
}
