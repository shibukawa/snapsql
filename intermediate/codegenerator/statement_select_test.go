package codegenerator

import (
	"strings"
	"testing"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSelectInstructions(t *testing.T) {
	tests := []struct {
		name                 string
		sql                  string
		dialect              snapsql.Dialect
		expectError          bool
		errorContains        string
		expectedInstructions []Instruction
		expectedCELCount     int
		expectedEnvCount     int
	}{
		{
			name:             "minimal select from",
			sql:              "SELECT id FROM users",
			dialect:          snapsql.DialectPostgres,
			expectError:      false,
			expectedCELCount: 0,
			expectedEnvCount: 0,
			expectedInstructions: []Instruction{
				// 最適化後: 連続する EMIT_STATIC はマージされる
				{Op: OpEmitStatic, Value: "SELECT id FROM users", Pos: "1:1"},
				// システムLIMIT命令（SQLに存在しない場合）
				{Op: OpIfSystemLimit, Pos: "0:0"},
				{Op: OpEmitStatic, Value: " LIMIT ", Pos: "0:0"},
				{Op: OpEmitSystemLimit, Pos: "0:0"},
				{Op: OpEnd, Pos: "0:0"},
				// システムOFFSET命令（SQLに存在しない場合）
				{Op: OpIfSystemOffset, Pos: "0:0"},
				{Op: OpEmitStatic, Value: " OFFSET ", Pos: "0:0"},
				{Op: OpEmitSystemOffset, Pos: "0:0"},
				{Op: OpEnd, Pos: "0:0"},
				// システムFOR命令（SQLに存在しない場合）
				{Op: OpEmitSystemFor, Pos: ""},
			},
		},
		{
			name:             "select multiple columns",
			sql:              "SELECT id, name FROM users",
			dialect:          "postgres",
			expectError:      false,
			expectedCELCount: 0,
			expectedEnvCount: 0,
			expectedInstructions: []Instruction{
				// 最適化後: 連続する EMIT_STATIC はマージされる
				{Op: OpEmitStatic, Value: "SELECT id, name FROM users", Pos: "1:1"},
				// システム命令
				{Op: OpIfSystemLimit, Pos: "0:0"},
				{Op: OpEmitStatic, Value: " LIMIT ", Pos: "0:0"},
				{Op: OpEmitSystemLimit, Pos: "0:0"},
				{Op: OpEnd, Pos: "0:0"},
				{Op: OpIfSystemOffset, Pos: "0:0"},
				{Op: OpEmitStatic, Value: " OFFSET ", Pos: "0:0"},
				{Op: OpEmitSystemOffset, Pos: "0:0"},
				{Op: OpEnd, Pos: "0:0"},
				// システムFOR命令（SQLに存在しない場合）
				{Op: OpEmitSystemFor, Pos: ""},
			},
		},
		{
			name:             "select with limit and offset",
			sql:              "SELECT id FROM users LIMIT 10 OFFSET 5",
			dialect:          "postgres",
			expectError:      false,
			expectedCELCount: 0,
			expectedEnvCount: 0,
			expectedInstructions: []Instruction{
				// 最適化後: 連続する EMIT_STATIC はマージされる
				// LIMIT句の前まで（スペースを含む） + LIMIT句の' LIMIT 'までマージ
				{Op: OpEmitStatic, Value: "SELECT id FROM users  LIMIT ", Pos: "1:1"},
				{Op: OpIfSystemLimit, Pos: "0:0"},
				{Op: OpEmitSystemLimit, Pos: "0:0"},
				{Op: OpElse, Pos: "0:0"},
				{Op: OpEmitStatic, Value: "10", Pos: "0:0"},
				{Op: OpEnd, Pos: "0:0"},
				// OFFSET句
				{Op: OpEmitStatic, Value: " OFFSET ", Pos: "0:0"},
				{Op: OpIfSystemOffset, Pos: "0:0"},
				{Op: OpEmitSystemOffset, Pos: "0:0"},
				{Op: OpElse, Pos: "0:0"},
				{Op: OpEmitStatic, Value: "5", Pos: "0:0"},
				{Op: OpEnd, Pos: "0:0"},
				// システムFOR命令（SQLに存在しない場合）
				{Op: OpEmitSystemFor, Pos: ""},
			},
		},
		{
			name:             "select with for update",
			sql:              "SELECT id FROM users FOR UPDATE",
			dialect:          "postgres",
			expectError:      false,
			expectedCELCount: 0,
			expectedEnvCount: 0,
			expectedInstructions: []Instruction{
				// 最適化後: 連続する EMIT_STATIC はマージされる
				{Op: OpEmitStatic, Value: "SELECT id FROM users ", Pos: "1:1"},
				// システムLIMIT命令
				{Op: OpIfSystemLimit, Pos: "0:0"},
				{Op: OpEmitStatic, Value: " LIMIT ", Pos: "0:0"},
				{Op: OpEmitSystemLimit, Pos: "0:0"},
				{Op: OpEnd, Pos: "0:0"},
				// システムOFFSET命令
				{Op: OpIfSystemOffset, Pos: "0:0"},
				{Op: OpEmitStatic, Value: " OFFSET ", Pos: "0:0"},
				{Op: OpEmitSystemOffset, Pos: "0:0"},
				{Op: OpEnd, Pos: "0:0"},
				// FOR句（SQLに存在する場合、単にEMIT_STATICで出力）
				{Op: OpEmitStatic, Value: " FOR UPDATE", Pos: "1:22"},
			},
		},
		{
			name:             "select with for share",
			sql:              "SELECT id FROM users FOR SHARE",
			dialect:          "postgres",
			expectError:      false,
			expectedCELCount: 0,
			expectedEnvCount: 0,
			expectedInstructions: []Instruction{
				// 最適化後: 連続する EMIT_STATIC はマージされる
				{Op: OpEmitStatic, Value: "SELECT id FROM users ", Pos: "1:1"},
				// システムLIMIT命令
				{Op: OpIfSystemLimit, Pos: "0:0"},
				{Op: OpEmitStatic, Value: " LIMIT ", Pos: "0:0"},
				{Op: OpEmitSystemLimit, Pos: "0:0"},
				{Op: OpEnd, Pos: "0:0"},
				// システムOFFSET命令
				{Op: OpIfSystemOffset, Pos: "0:0"},
				{Op: OpEmitStatic, Value: " OFFSET ", Pos: "0:0"},
				{Op: OpEmitSystemOffset, Pos: "0:0"},
				{Op: OpEnd, Pos: "0:0"},
				// FOR句（SQLに存在する場合、単にEMIT_STATICで出力）
				{Op: OpEmitStatic, Value: " FOR SHARE", Pos: "1:22"},
			},
		},
		{
			name:             "select with for update nowait",
			sql:              "SELECT id FROM users FOR UPDATE NOWAIT",
			dialect:          "postgres",
			expectError:      false,
			expectedCELCount: 0,
			expectedEnvCount: 0,
			expectedInstructions: []Instruction{
				// 最適化後: 連続する EMIT_STATIC はマージされる
				{Op: OpEmitStatic, Value: "SELECT id FROM users ", Pos: "1:1"},
				// システムLIMIT命令
				{Op: OpIfSystemLimit, Pos: "0:0"},
				{Op: OpEmitStatic, Value: " LIMIT ", Pos: "0:0"},
				{Op: OpEmitSystemLimit, Pos: "0:0"},
				{Op: OpEnd, Pos: "0:0"},
				// システムOFFSET命令
				{Op: OpIfSystemOffset, Pos: "0:0"},
				{Op: OpEmitStatic, Value: " OFFSET ", Pos: "0:0"},
				{Op: OpEmitSystemOffset, Pos: "0:0"},
				{Op: OpEnd, Pos: "0:0"},
				// FOR句（SQLに存在する場合、単にEMIT_STATICで出力）
				{Op: OpEmitStatic, Value: " FOR UPDATE NOWAIT", Pos: "1:22"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// parser.ParseSQLFile でパース
			reader := strings.NewReader(tt.sql)
			stmt, _, err := parser.ParseSQLFile(reader, nil, "", "", parser.Options{})
			require.NoError(t, err, "ParseSQLFile should succeed")
			require.NotNil(t, stmt, "statement should not be nil")

			// GenerationContext を作成
			ctx := NewGenerationContext(tt.dialect)

			// 命令列を生成
			instructions, celExpressions, celEnvironments, err := GenerateSelectInstructions(stmt, ctx)

			if tt.expectError {
				require.Error(t, err)

				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}

				return
			}

			require.NoError(t, err, "GenerateSelectInstructions should succeed")

			// CEL式と環境の検証
			assert.Len(t, celExpressions, tt.expectedCELCount, "CEL expressions count mismatch")
			assert.Len(t, celEnvironments, tt.expectedEnvCount, "CEL environments count mismatch")

			// 命令列全体をdeep equalで検証
			assert.Equal(t, tt.expectedInstructions, instructions, "Instructions mismatch")
		})
	}
}
