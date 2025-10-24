package codegenerator

import (
	"errors"
	"fmt"

	"github.com/shibukawa/snapsql/parser"
)

// generateUpdateClause は UPDATE 節から命令を生成する
//
// UPDATE節の構造:
//
//	UPDATE table_name
//
// Parameters:
//   - clause: *parser.UpdateClause（必須）
//   - builder: *InstructionBuilder
//
// Returns:
//   - error: エラー
func generateUpdateClause(clause *parser.UpdateClause, builder *InstructionBuilder, skipLeadingTrivia bool) error {
	if clause == nil {
		return errors.New("UPDATE clause is required")
	}

	// UPDATE トークンを処理
	tokens := clause.RawTokens()
	options := []ProcessTokensOption{}
	if skipLeadingTrivia {
		options = append(options, WithSkipLeadingTrivia())
	}
	if err := builder.ProcessTokens(tokens, options...); err != nil {
		return fmt.Errorf("code generation: %w", err)
	}

	return nil
}
