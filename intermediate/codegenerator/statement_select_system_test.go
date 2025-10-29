package codegenerator

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
)

// TestSystemLimitOffsetInstructions はLIMIT/OFFSETのシステム命令生成を詳細にテストする
func TestSystemLimitOffsetInstructions(t *testing.T) {
	tests := []struct {
		name                    string
		sql                     string
		expectedLimitDefault    string // "" の場合はLIMIT句が存在しない
		expectedOffsetDefault   string // "" の場合はOFFSET句が存在しない
		expectLimitKeywordInIf  bool   // LIMIT キーワードが IF ブロック内にあるか
		expectOffsetKeywordInIf bool   // OFFSET キーワードが IF ブロック内にあるか
	}{
		{
			name:                    "no LIMIT and no OFFSET",
			sql:                     "SELECT id FROM users",
			expectedLimitDefault:    "",
			expectedOffsetDefault:   "",
			expectLimitKeywordInIf:  true,
			expectOffsetKeywordInIf: true,
		},
		{
			name:                    "with LIMIT only",
			sql:                     "SELECT id FROM users LIMIT 10",
			expectedLimitDefault:    "10",
			expectedOffsetDefault:   "",
			expectLimitKeywordInIf:  false,
			expectOffsetKeywordInIf: true,
		},
		{
			name:                    "with OFFSET only (should not happen in SQL)",
			sql:                     "SELECT id FROM users OFFSET 5",
			expectedLimitDefault:    "",
			expectedOffsetDefault:   "5",
			expectLimitKeywordInIf:  true,
			expectOffsetKeywordInIf: false,
		},
		{
			name:                    "with both LIMIT and OFFSET",
			sql:                     "SELECT id FROM users LIMIT 10 OFFSET 5",
			expectedLimitDefault:    "10",
			expectedOffsetDefault:   "5",
			expectLimitKeywordInIf:  false,
			expectOffsetKeywordInIf: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// パース
			reader := strings.NewReader(tt.sql)

			stmt, _, _, err := parser.ParseSQLFile(reader, nil, "", "", parser.Options{})
			if err != nil {
				t.Fatalf("ParseSQLFile failed: %v", err)
			}

			// 命令生成
			ctx := NewGenerationContext(snapsql.DialectPostgres)

			instructions, _, _, err := GenerateSelectInstructions(stmt, ctx)
			if err != nil {
				t.Fatalf("GenerateSelectInstructions failed: %v", err)
			}

			// デバッグ出力
			t.Logf("Generated %d instructions:", len(instructions))

			for i, inst := range instructions {
				data, _ := json.Marshal(inst)
				t.Logf("  [%d] %s", i, string(data))
			}

			// LIMIT関連の検証
			checkSystemBlock(t, instructions, "LIMIT", tt.expectedLimitDefault, tt.expectLimitKeywordInIf)

			// OFFSET関連の検証
			checkSystemBlock(t, instructions, "OFFSET", tt.expectedOffsetDefault, tt.expectOffsetKeywordInIf)
		})
	}
}

func checkSystemBlock(t *testing.T, instructions []Instruction, keyword string, expectedDefault string, expectKeywordInIf bool) {
	t.Helper()

	var (
		ifOp   string
		emitOp string
	)

	if keyword == "LIMIT" {
		ifOp = OpIfSystemLimit
		emitOp = OpEmitSystemLimit
	} else {
		ifOp = OpIfSystemOffset
		emitOp = OpEmitSystemOffset
	}

	// IF_SYSTEM_* 命令を探す
	foundIf := false
	foundEmit := false
	foundKeywordInIf := false
	foundDefault := false

	inIfBlock := false

	for _, inst := range instructions {
		if inst.Op == ifOp {
			foundIf = true
			inIfBlock = true
		}

		if inIfBlock && inst.Op == OpEmitStatic && strings.Contains(inst.Value, keyword) {
			foundKeywordInIf = true
		}

		if inst.Op == emitOp {
			foundEmit = true
		}

		if inst.Op == OpEmitStatic && inst.Value == expectedDefault && expectedDefault != "" {
			foundDefault = true
		}

		if inst.Op == OpEnd {
			inIfBlock = false
		}
	}

	if !foundIf {
		t.Errorf("%s: IF_SYSTEM_%s not found", keyword, keyword)
	}

	if !foundEmit {
		t.Errorf("%s: EMIT_SYSTEM_%s not found", keyword, keyword)
	}

	if expectKeywordInIf != foundKeywordInIf {
		t.Errorf("%s: expected keyword in IF block=%v, got=%v", keyword, expectKeywordInIf, foundKeywordInIf)
	}

	if expectedDefault != "" && !foundDefault {
		t.Errorf("%s: default value '%s' not found", keyword, expectedDefault)
	}
}
