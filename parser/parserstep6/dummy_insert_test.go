package parserstep6

import (
	"testing"

	"github.com/shibukawa/snapsql/parser/parserstep1"
	"github.com/shibukawa/snapsql/tokenizer"
	"github.com/stretchr/testify/assert"
)

// このテストでは、parserstep1までの処理を行い、トークンの状態を確認します
func TestInsertWithDummyLiterals(t *testing.T) {
	tests := []struct {
		name   string
		sql    string
		params map[string]any
	}{
		{
			name: "INSERT with dummy literals",
			sql:  "INSERT INTO users (id, name) VALUES (/*= id */, /*= name */)",
			params: map[string]any{
				"id":   1,
				"name": "John",
			},
		},
		{
			name: "INSERT with const literals",
			sql:  "INSERT INTO users (id, name) VALUES (/*$ id */, /*$ name */)",
			params: map[string]any{
				"id":   1,
				"name": "John",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// トークナイズ
			tokens, err := tokenizer.Tokenize(tt.sql)
			assert.NoError(t, err)

			// デバッグ用：トークンを表示
			t.Log("Original Tokens:")
			for i, token := range tokens {
				t.Logf("[%d] %s: %s", i, token.Type, token.Value)
			}

			// Step 1: Run parserstep1 - Basic syntax validation and dummy literal insertion
			processedTokens, err := parserstep1.Execute(tokens)
			assert.NoError(t, err)

			// デバッグ用：処理後のトークンを表示
			t.Log("Processed Tokens after parserstep1:")
			for i, token := range processedTokens {
				t.Logf("[%d] %s: %s", i, token.Type, token.Value)
			}

			// 手動でダミーリテラルを挿入（parserstep2以降を実行せずに）
			var modifiedTokens []tokenizer.Token
			for _, token := range processedTokens {
				modifiedTokens = append(modifiedTokens, token)
				
				if token.Type == tokenizer.BLOCK_COMMENT {
					if token.Value == "/*= id */" {
						// ダミーリテラルを挿入
						modifiedTokens = append(modifiedTokens, tokenizer.Token{
							Type:  tokenizer.DUMMY_START,
							Value: "DUMMY_START",
						})
						modifiedTokens = append(modifiedTokens, tokenizer.Token{
							Type:  tokenizer.NUMBER,
							Value: "1",
						})
						modifiedTokens = append(modifiedTokens, tokenizer.Token{
							Type:  tokenizer.DUMMY_END,
							Value: "DUMMY_END",
						})
					} else if token.Value == "/*= name */" {
						// ダミーリテラルを挿入
						modifiedTokens = append(modifiedTokens, tokenizer.Token{
							Type:  tokenizer.DUMMY_START,
							Value: "DUMMY_START",
						})
						modifiedTokens = append(modifiedTokens, tokenizer.Token{
							Type:  tokenizer.STRING,
							Value: "'John'",
						})
						modifiedTokens = append(modifiedTokens, tokenizer.Token{
							Type:  tokenizer.DUMMY_END,
							Value: "DUMMY_END",
						})
					} else if token.Value == "/*$ id */" {
						// 定数リテラルを挿入
						modifiedTokens = append(modifiedTokens, tokenizer.Token{
							Type:  tokenizer.NUMBER,
							Value: "1",
						})
					} else if token.Value == "/*$ name */" {
						// 定数リテラルを挿入
						modifiedTokens = append(modifiedTokens, tokenizer.Token{
							Type:  tokenizer.STRING,
							Value: "'John'",
						})
					}
				}
			}

			// デバッグ用：手動で修正したトークンを表示
			t.Log("Manually Modified Tokens:")
			for i, token := range modifiedTokens {
				t.Logf("[%d] %s: %s", i, token.Type, token.Value)
			}

			// 手動で修正したトークンでparserstep1を再実行
			finalTokens, err := parserstep1.Execute(modifiedTokens)
			if err != nil {
				t.Logf("Error after manual modification: %v", err)
			} else {
				t.Log("Final Tokens after parserstep1:")
				for i, token := range finalTokens {
					t.Logf("[%d] %s: %s", i, token.Type, token.Value)
				}
			}
		})
	}
}
