package codegenerator

import (
	"fmt"

	"github.com/shibukawa/snapsql/parser"
)

// generateSelectClause は SELECT 句から命令列を生成する
func generateSelectClause(clause *parser.SelectClause, builder *InstructionBuilder, skipLeadingTrivia bool) error {
	if clause == nil {
		return fmt.Errorf("%w: SELECT clause is nil", ErrClauseNil)
	}

	tokens := clause.RawTokens()

	// Phase 1: トークンをそのまま処理
	// 将来的には、ここでトークンのカスタマイズを行う
	// 例: SELECT DISTINCT の処理、集約関数の特別処理など

	options := []ProcessTokensOption{}
	if skipLeadingTrivia {
		options = append(options, WithSkipLeadingTrivia())
	}

	if err := builder.ProcessTokens(tokens, options...); err != nil {
		return fmt.Errorf("failed to process tokens in SELECT clause: %w", err)
	}

	return nil
}
