package codegenerator

import (
	"fmt"

	"github.com/shibukawa/snapsql/parser"
)

// generateSetClause は SET 節から命令を生成する
//
// SET節の構造:
//   SET column1 = value1, column2 = value2, ...
//
// Parameters:
//   - clause: *parser.SetClause（必須）
//   - builder: *InstructionBuilder
//
// Returns:
//   - error: エラー
func generateSetClause(clause *parser.SetClause, builder *InstructionBuilder) error {
	if clause == nil {
		return fmt.Errorf("SET clause is required")
	}

	// SET トークンを処理
	tokens := clause.RawTokens()
	if err := builder.ProcessTokens(tokens); err != nil {
		return fmt.Errorf("code generation: %w", err)
	}

	return nil
}
