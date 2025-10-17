package codegenerator

import (
	"fmt"

	"github.com/shibukawa/snapsql/parser"
)

// GenerateUpdateInstructions は UPDATE 文から命令列と CEL 式を生成する
//
// Parameters:
//   - stmt: parser.StatementNode (内部で *parser.UpdateStatement にキャスト)
//   - ctx: GenerationContext（方言、テーブル情報等）
//
// Returns:
//   - []Instruction: 生成された命令列
//   - []CELExpression: CEL 式のリスト
//   - []CELEnvironment: CEL 環境のリスト
//   - error: エラー
func GenerateUpdateInstructions(stmt parser.StatementNode, ctx *GenerationContext) ([]Instruction, []CELExpression, []CELEnvironment, error) {
	updateStmt, ok := stmt.(*parser.UpdateStatement)
	if !ok {
		return nil, nil, nil, fmt.Errorf("%w: expected *parser.UpdateStatement, got %T", ErrStatementTypeMismatch, stmt)
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
	if updateStmt.CTE() != nil {
		if err := generateCTEClause(updateStmt.CTE(), builder); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to generate CTE clause: %w", err)
		}
	}

	// Phase 2: UPDATE 句を処理（必須）
	if err := generateUpdateClause(updateStmt.Update, builder); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate UPDATE clause: %w", err)
	}

	// Phase 3: SET 句を処理（必須）
	if err := generateSetClause(updateStmt.Set, builder); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate SET clause: %w", err)
	}

	// Phase 4: WHERE 句を処理（任意）
	if updateStmt.Where != nil {
		if err := generateWhereClause(updateStmt.Where, builder); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to generate WHERE clause: %w", err)
		}
	}

	// Phase 5: RETURNING 句を処理（任意）
	if updateStmt.Returning != nil {
		if err := generateReturningClause(updateStmt.Returning, builder); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to generate RETURNING clause: %w", err)
		}
	}

	// 命令列、CEL式、CEL環境を取得
	instructions := builder.Finalize()
	celExpressions := ctx.Expressions
	celEnvironments := builder.GetCELEnvironments()

	return instructions, celExpressions, celEnvironments, nil
}
