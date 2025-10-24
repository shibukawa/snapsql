package codegenerator

import (
	"errors"
	"fmt"

	"github.com/shibukawa/snapsql/parser"
)

// generateDeleteFromClause は DELETE FROM 節から命令を生成する
//
// DELETE FROM節の構造:
//
//	DELETE FROM table_name
//
// Parameters:
//   - clause: *parser.DeleteFromClause（必須）
//   - builder: *InstructionBuilder
//
// Returns:
//   - error: エラー
func generateDeleteFromClause(clause *parser.DeleteFromClause, builder *InstructionBuilder, skipLeadingTrivia bool) error {
	if clause == nil {
		return errors.New("DELETE FROM clause is required")
	}

	// DELETE FROM トークンを処理
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
