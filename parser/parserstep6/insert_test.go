package parserstep6

import (
	"testing"

	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/parser/parserstep1"
	"github.com/shibukawa/snapsql/parser/parserstep2"
	"github.com/shibukawa/snapsql/parser/parserstep3"
	"github.com/shibukawa/snapsql/parser/parserstep4"
	"github.com/shibukawa/snapsql/parser/parserstep5"
	"github.com/shibukawa/snapsql/tokenizer"
	"github.com/stretchr/testify/assert"
)

// parseUpToStep5 は parser.RawParse の一部を再実装し、parserstep5までを実行します
func parseUpToStep5ForInsert(sql string) (cmn.StatementNode, error) {
	// トークナイズ
	tokens, err := tokenizer.Tokenize(sql)
	if err != nil {
		return nil, err
	}

	// デバッグ用：トークンを表示
	// fmt.Println("Tokens:")
	// for i, token := range tokens {
	//     fmt.Printf("[%d] %s: %s\n", i, token.Type, token.Value)
	// }

	// Step 1: Run parserstep1 - Basic syntax validation and dummy literal insertion
	processedTokens, err := parserstep1.Execute(tokens)
	if err != nil {
		return nil, err
	}

	// デバッグ用：処理後のトークンを表示
	// fmt.Println("Processed Tokens:")
	// for i, token := range processedTokens {
	//     fmt.Printf("[%d] %s: %s\n", i, token.Type, token.Value)
	// }

	// Step 2: Run parserstep2 - SQL structure parsing
	stmt, err := parserstep2.Execute(processedTokens)
	if err != nil {
		return nil, err
	}

	// Step 3: Run parserstep3 - Clause-level validation and assignment
	if err := parserstep3.Execute(stmt); err != nil {
		return nil, err
	}

	// Step 4: Run parserstep4 - Clause content validation
	if err := parserstep4.Execute(stmt); err != nil {
		return nil, err
	}

	// Step 5: Run parserstep5 - Directive structure validation
	if err := parserstep5.Execute(stmt); err != nil {
		return nil, err
	}

	return stmt, nil
}

func TestInsertParsing(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		wantErr bool
	}{
		{
			name:    "Basic INSERT with column list",
			sql:     "INSERT INTO users (id, name) VALUES (1, 'John')",
			wantErr: false,
		},
		{
			name:    "INSERT with column list and comment",
			sql:     "INSERT INTO users (id, name) VALUES (/*= id */, 'John')",
			wantErr: false,
		},
		{
			name:    "INSERT without column list",
			sql:     "INSERT INTO users VALUES (1, 'John')",
			wantErr: true, // 期待されるエラー
		},
		{
			name:    "INSERT with SELECT",
			sql:     "INSERT INTO users SELECT id, name FROM temp_users",
			wantErr: false,
		},
		{
			name:    "INSERT with dummy values",
			sql:     "INSERT INTO users (id, name) VALUES (1, /*= name */)",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseUpToStep5ForInsert(tt.sql)
			if tt.wantErr {
				assert.Error(t, err)
				t.Logf("Expected error: %v", err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
