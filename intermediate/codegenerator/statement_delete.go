package codegenerator

import (
	"fmt"

	"github.com/shibukawa/snapsql/parser"
)

// GenerateDeleteInstructions は DELETE 文から命令列と CEL 式を生成する
//
// Parameters:
//   - stmt: parser.StatementNode (内部で *parser.DeleteFromStatement にキャスト)
//   - ctx: GenerationContext（方言、テーブル情報等）
//
// Returns:
//   - []Instruction: 生成された命令列
//   - []CELExpression: CEL 式のリスト
//   - []CELEnvironment: CEL 環境のリスト
//   - error: エラー
func GenerateDeleteInstructions(stmt parser.StatementNode, ctx *GenerationContext) ([]Instruction, []CELExpression, []CELEnvironment, error) {
	deleteStmt, ok := stmt.(*parser.DeleteFromStatement)
	if !ok {
		return nil, nil, nil, fmt.Errorf("%w: expected *parser.DeleteFromStatement, got %T", ErrStatementTypeMismatch, stmt)
	}

	// Root 環境がなければ作成（最初の呼び出しのみ）
	if len(ctx.CELEnvironments) == 0 {
		ctx.AddCELEnvironment(CELEnvironment{
			Container:   "root",
			ParentIndex: nil,
		})
	}

	builder := NewInstructionBuilder(ctx)

	// Phase 1: CTE（WITH句）を処理（任意）
	if deleteStmt.CTE() != nil {
		if err := generateCTEClause(deleteStmt.CTE(), builder); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to generate CTE clause: %w", err)
		}
	}

	// Phase 2: DELETE FROM 句を処理（必須）
	if err := generateDeleteFromClause(deleteStmt.From, builder); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate DELETE FROM clause: %w", err)
	}

	// Phase 3: WHERE 句を処理（任意）
	if deleteStmt.Where != nil {
		if err := generateWhereClause(deleteStmt.Where, builder); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to generate WHERE clause: %w", err)
		}
	}

	// Phase 4: RETURNING 句を処理（任意）
	if deleteStmt.Returning != nil {
		if err := generateReturningClause(deleteStmt.Returning, builder); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to generate RETURNING clause: %w", err)
		}
	}

	// 命令列、CEL式、CEL環境を取得
	instructions := builder.Finalize()
	celExpressions := ctx.Expressions
	celEnvironments := builder.GetCELEnvironments()

	return instructions, celExpressions, celEnvironments, nil
}
