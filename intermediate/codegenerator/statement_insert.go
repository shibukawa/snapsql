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
	insertStmt, ok := stmt.(*parser.InsertIntoStatement)
	if !ok {
		return nil, nil, nil, fmt.Errorf("%w: expected *parser.InsertIntoStatement, got %T", ErrStatementTypeMismatch, stmt)
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
	if insertStmt.CTE() != nil {
		if err := generateCTEClause(insertStmt.CTE(), builder); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to generate CTE clause: %w", err)
		}
	}

	// INSERT INTO 句を処理（必須）
	if err := generateInsertIntoClause(insertStmt.Into, insertStmt.Columns, builder); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate INSERT INTO clause: %w", err)
	}

	// VALUES句とSELECT句の分岐処理
	if insertStmt.ValuesList != nil {
		// VALUES形式のINSERT
		if err := generateValuesClause(insertStmt.ValuesList, builder, insertStmt.Columns); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to generate VALUES clause: %w", err)
		}
	} else if insertStmt.Select != nil {
		// INSERT ... SELECT形式
		if err := generateSelectClause(insertStmt.Select, builder); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to generate SELECT clause: %w", err)
		}

		// FROM句以降を処理（SELECT文と同様）
		if insertStmt.From != nil {
			if err := generateFromClause(insertStmt.From, builder); err != nil {
				return nil, nil, nil, fmt.Errorf("failed to generate FROM clause: %w", err)
			}
		}

		if insertStmt.Where != nil {
			if err := generateWhereClause(insertStmt.Where, builder); err != nil {
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
			if err := generateLimitClause(insertStmt.Limit, builder); err != nil {
				return nil, nil, nil, fmt.Errorf("failed to generate LIMIT clause: %w", err)
			}
		}

		if insertStmt.Offset != nil {
			if err := generateOffsetClause(insertStmt.Offset, builder); err != nil {
				return nil, nil, nil, fmt.Errorf("failed to generate OFFSET clause: %w", err)
			}
		}
	} else {
		// VALUES も SELECT も存在しない場合はエラー
		return nil, nil, nil, fmt.Errorf("%w: either VALUES or SELECT clause is required for INSERT", ErrMissingClause)
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
