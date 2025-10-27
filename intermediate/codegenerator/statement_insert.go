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

	// Add parameters from FunctionDefinition to root environment if available
	// The root environment (index 0) is already created by NewGenerationContext,
	// so we update it with parameters information
	if funcDef != nil && funcDef.ParameterOrder != nil {
		if len(ctx.CELEnvironments) > 0 {
			// Update existing root environment with parameters
			for _, paramName := range funcDef.ParameterOrder {
				originalParamValue := funcDef.OriginalParameters[paramName]
				if originalParamValue != nil {
					// Extract parameter type
					paramType := extractParameterTypeString(originalParamValue)
					ctx.CELEnvironments[0].AdditionalVariables = append(
						ctx.CELEnvironments[0].AdditionalVariables,
						CELVariableInfo{
							Name: paramName,
							Type: paramType,
						},
					)
				}
			}
		} else {
			// Fallback: Create root environment if not exists
			rootEnv := CELEnvironment{
				Container:           "root",
				ParentIndex:         nil,
				AdditionalVariables: make([]CELVariableInfo, 0),
			}

			// Add parameters
			for _, paramName := range funcDef.ParameterOrder {
				originalParamValue := funcDef.OriginalParameters[paramName]
				if originalParamValue != nil {
					paramType := extractParameterTypeString(originalParamValue)
					rootEnv.AdditionalVariables = append(rootEnv.AdditionalVariables, CELVariableInfo{
						Name: paramName,
						Type: paramType,
					})
				}
			}

			ctx.AddCELEnvironment(rootEnv)
		}
	}

	builder := NewInstructionBuilder(ctx)

	// Phase 4: CTE（WITH句）を処理
	if insertStmt.CTE() != nil {
		if err := generateCTEClause(insertStmt.CTE(), builder); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to generate CTE clause: %w", err)
		}
	}

	// INSERT INTO 句を処理（必須）
	skipLeadingInto := insertStmt.CTE() == nil
	if err := generateInsertIntoClause(insertStmt.Into, insertStmt.Columns, builder, skipLeadingInto); err != nil {
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
		skipLeading := insertStmt.CTE() == nil
		if err := generateSelectClause(insertStmt.Select, builder, skipLeading); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to generate SELECT clause: %w", err)
		}

		// Get system fields that need to be added to SELECT
		existingColumns := make(map[string]bool)
		for _, col := range insertStmt.Columns {
			existingColumns[col.Name] = true
		}
		systemFields := getInsertSystemFieldsFiltered(ctx, existingColumns)

		// Append system field expressions to SELECT clause
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

// extractParameterTypeString extracts the type from a parameter value
// This is used to populate the CEL environment with parameter type information
func extractParameterTypeString(paramValue any) string {
	// The parameter value could be a string (simple type) or a map (complex type definition)
	switch v := paramValue.(type) {
	case string:
		// Simple type like "int", "string", "bool", etc.
		return v
	case []any:
		// Array type: the element type is the first item
		// For [{ id: int, name: string }], extract the object element type and append []
		if len(v) > 0 {
			elementType := extractParameterTypeString(v[0])
			return "[" + elementType + "]"
		}
		// Empty array, default to "any[]"
		return "[]"
	case map[string]any:
		// Complex type definition: could have "type" field or be a direct object structure
		if typeVal, ok := v["type"]; ok {
			if typeStr, ok := typeVal.(string); ok {
				return typeStr
			}
		}
		// If no "type" field, this is a direct object structure like { id: int, name: string }
		// Return a representation that can be recognized as an object type
		// by checking for the presence of fields like "id", "name", etc.
		// For now, return a generic object type indicator
		return "{ " + mapToTypeString(v) + " }"
	default:
		// Unknown type, fallback to "any"
		return "any"
	}
}

// mapToTypeString converts a map to a type string representation like "id: int, name: string"
func mapToTypeString(m map[string]any) string {
	if len(m) == 0 {
		return ""
	}
	// For simplicity, just return a indicator that this is an object
	// The actual field types are not critical here, just the fact that it's an object
	return "..."
}
