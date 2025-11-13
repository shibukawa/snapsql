package codegenerator

import (
	"fmt"

	"github.com/shibukawa/snapsql/parser"
)

// GenerateInsertInstructions は INSERT 文から命令列と CEL 式を生成する
//
// Parameters:
//   - stmt: parser.StatementNode (内部で *parser.InsertIntoStatement にキャスト)
//   - ctx: GenerationContext（方言、テーブル情報等）
//
// Returns:
//   - []Instruction: 生成された命令列
//   - []CELExpression: CEL 式のリスト
//   - []CELEnvironment: CEL 環境のリスト
//   - error: エラー
func GenerateInsertInstructions(stmt parser.StatementNode, ctx *GenerationContext) ([]Instruction, []CELExpression, []CELEnvironment, error) {
	return GenerateInsertInstructionsWithFunctionDef(stmt, ctx, nil)
}

// GenerateInsertInstructionsWithFunctionDef は INSERT 文から命令列と CEL 式を生成する
// FunctionDefinition を受け取り、parameter 情報を CELEnvironment に追加する
//
// Parameters:
//   - stmt: parser.StatementNode (内部で *parser.InsertIntoStatement にキャスト)
//   - ctx: GenerationContext（方言、テーブル情報等）
//   - funcDef: *parser.FunctionDefinition（parameter 情報を含む、nil 可能）
//
// Returns:
//   - []Instruction: 生成された命令列
//   - []CELExpression: CEL 式のリスト
//   - []CELEnvironment: CEL 環境のリスト
//   - error: エラー
func GenerateInsertInstructionsWithFunctionDef(stmt parser.StatementNode, ctx *GenerationContext, funcDef *parser.FunctionDefinition) ([]Instruction, []CELExpression, []CELEnvironment, error) {
	insertStmt, ok := stmt.(*parser.InsertIntoStatement)
	if !ok {
		return nil, nil, nil, fmt.Errorf("%w: expected *parser.InsertIntoStatement, got %T", ErrStatementTypeMismatch, stmt)
	}

	// Store statement in context for later reference (e.g., column access in handleObjectExpansion)
	ctx.Statement = stmt

	builder := NewInstructionBuilder(ctx)

	// Phase 4: CTE（WITH句）を処理
	if insertStmt.CTE() != nil {
		if err := generateCTEClause(insertStmt.CTE(), builder); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to generate CTE clause: %w", err)
		}
	}

	// INSERT INTO 句を処理（必須）
	skipLeadingInto := insertStmt.CTE() == nil

	systemFields, err := generateInsertIntoClause(insertStmt.Into, insertStmt.Columns, builder, skipLeadingInto)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate INSERT INTO clause: %w", err)
	}

	// Add system fields to Columns if they were added by generateInsertIntoClause
	// This ensures that generateValuesClause can detect them
	if len(systemFields) > 0 {
		for _, field := range systemFields {
			insertStmt.Columns = append(insertStmt.Columns, parser.FieldName{Name: field.Name})
		}
	}

	// VALUES句とSELECT句の分岐処理
	if insertStmt.ValuesList != nil {
		// VALUES形式のINSERT
		if err := generateValuesClause(insertStmt.ValuesList, builder, insertStmt.Columns, systemFields); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to generate VALUES clause: %w", err)
		}
	} else if insertStmt.Select != nil {
		// INSERT ... SELECT形式
		skipLeading := insertStmt.CTE() == nil
		if err := generateSelectClause(insertStmt.Select, builder, skipLeading); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to generate SELECT clause: %w", err)
		}

		// Append system field expressions to SELECT clause
		// Use the system fields that were added by generateInsertIntoClause
		if len(systemFields) > 0 {
			appendSystemFieldsToSelectClause(builder, systemFields)
		}

		// FROM句以降を処理（SELECT文と同様）
		if insertStmt.From != nil {
			if err := generateFromClause(insertStmt.From, builder); err != nil {
				return nil, nil, nil, fmt.Errorf("failed to generate FROM clause: %w", err)
			}
		}

		if insertStmt.Where != nil {
			if _, err := generateWhereClause(insertStmt.Where, builder, false); err != nil {
				return nil, nil, nil, fmt.Errorf("failed to generate WHERE clause: %w", err)
			}
		}

		if insertStmt.GroupBy != nil {
			if err := generateGroupByClause(insertStmt.GroupBy, builder); err != nil {
				return nil, nil, nil, fmt.Errorf("failed to generate GROUP BY clause: %w", err)
			}
		}

		if insertStmt.Having != nil {
			if err := generateHavingClause(insertStmt.Having, builder); err != nil {
				return nil, nil, nil, fmt.Errorf("failed to generate HAVING clause: %w", err)
			}
		}

		if insertStmt.OrderBy != nil {
			if err := generateOrderByClause(insertStmt.OrderBy, builder); err != nil {
				return nil, nil, nil, fmt.Errorf("failed to generate ORDER BY clause: %w", err)
			}
		}

		if insertStmt.Limit != nil {
			if err := GenerateLimitClauseOrSystem(insertStmt.Limit, builder); err != nil {
				return nil, nil, nil, fmt.Errorf("failed to generate LIMIT clause: %w", err)
			}
		}

		if insertStmt.Offset != nil {
			if err := GenerateOffsetClauseOrSystem(insertStmt.Offset, builder); err != nil {
				return nil, nil, nil, fmt.Errorf("failed to generate OFFSET clause: %w", err)
			}
		}
	}

	// ON CONFLICT句を処理（PostgreSQL）
	if insertStmt.OnConflict != nil {
		if err := generateOnConflictClause(insertStmt.OnConflict, builder); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to generate ON CONFLICT clause: %w", err)
		}
	}

	// RETURNING句を処理
	if insertStmt.Returning != nil {
		if err := generateReturningClause(insertStmt.Returning, builder); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to generate RETURNING clause: %w", err)
		}
	}

	// 最適化と結果の取得
	instructions := builder.Finalize()
	celExpressions := ctx.Expressions

	celEnvironments := builder.GetCELEnvironments()

	return instructions, celExpressions, celEnvironments, nil
}
