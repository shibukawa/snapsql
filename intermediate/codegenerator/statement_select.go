package codegenerator

import (
	"fmt"

	"github.com/shibukawa/snapsql/parser"
)

// GenerateSelectInstructions は SELECT 文から命令列と CEL 式を生成する
//
// Parameters:
//   - stmt: parser.StatementNode (内部で *parser.SelectStatement にキャスト)
//   - ctx: GenerationContext（方言、テーブル情報等）
//
// Returns:
//   - []Instruction: 生成された命令列
//   - []CELExpression: CEL 式のリスト（Phase 1 では空）
//   - []CELEnvironment: CEL 環境のリスト（Phase 1 では空）
//   - error: エラー
func GenerateSelectInstructions(stmt parser.StatementNode, ctx *GenerationContext) ([]Instruction, []CELExpression, []CELEnvironment, error) {
	selectStmt, ok := stmt.(*parser.SelectStatement)
	if !ok {
		return nil, nil, nil, fmt.Errorf("%w: expected *parser.SelectStatement, got %T", ErrStatementTypeMismatch, stmt)
	}

	// Root 環境がなければ作成（最初の呼び出しのみ）
	if len(ctx.CELEnvironments) == 0 {
		ctx.AddCELEnvironment(CELEnvironment{
			Container:   "root",
			ParentIndex: nil,
		})
	}

	builder := NewInstructionBuilder(ctx)

	// Phase 4: CTE（WITH句）を処理
	if selectStmt.CTE() != nil {
		if err := generateCTEClause(selectStmt.CTE(), builder); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to generate CTE clause: %w", err)
		}
	}

	// SELECT 句を処理（必須）
	skipLeading := selectStmt.CTE() == nil
	if err := generateSelectClause(selectStmt.Select, builder, skipLeading); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate SELECT clause: %w", err)
	}

	// FROM 句を処理（必須）
	if err := generateFromClause(selectStmt.From, builder); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate FROM clause: %w", err)
	}

	// WHERE 句を処理（任意）
	if selectStmt.Where != nil {
		if err := generateWhereClause(selectStmt.Where, builder); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to generate WHERE clause: %w", err)
		}
	}

	// GROUP BY 句を処理（任意）
	if selectStmt.GroupBy != nil {
		if err := generateGroupByClause(selectStmt.GroupBy, builder); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to generate GROUP BY clause: %w", err)
		}
	}

	// HAVING 句を処理（任意）
	if selectStmt.Having != nil {
		if err := generateHavingClause(selectStmt.Having, builder); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to generate HAVING clause: %w", err)
		}
	}

	// ORDER BY 句を処理（任意）
	if selectStmt.OrderBy != nil {
		if err := generateOrderByClause(selectStmt.OrderBy, builder); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to generate ORDER BY clause: %w", err)
		}
	}

	// LIMIT 句を処理（任意）
	// GenerateLimitClauseOrSystem が nil と非 nil の両方のケースを処理
	if err := GenerateLimitClauseOrSystem(selectStmt.Limit, builder); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate LIMIT clause: %w", err)
	}

	// OFFSET 句を処理（任意）
	// GenerateOffsetClauseOrSystem が nil と非 nil の両方のケースを処理
	if err := GenerateOffsetClauseOrSystem(selectStmt.Offset, builder); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate OFFSET clause: %w", err)
	}

	// FOR 句を処理（任意）- 行ロック句
	// GenerateForClauseOrSystem が nil と非 nil の両方のケースを処理
	if err := GenerateForClauseOrSystem(selectStmt.For, builder); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate FOR clause: %w", err)
	}

	// システム命令の追加は各clause関数内で実施済み

	// 最適化と結果の取得
	instructions := builder.Finalize()
	// ctx.Expressions は既に CELExpression 型の配列
	celExpressions := ctx.Expressions

	celEnvironments := builder.GetCELEnvironments()

	return instructions, celExpressions, celEnvironments, nil
}
