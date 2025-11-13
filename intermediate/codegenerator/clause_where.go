package codegenerator

import (
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql/parser"
)

// generateWhereClause は WHERE 句から命令列を生成し、メタデータを返す
func generateWhereClause(clause *parser.WhereClause, builder *InstructionBuilder, critical bool) (*WhereClauseMeta, error) {
	if clause == nil {
		return nil, fmt.Errorf("%w: WHERE clause is nil", ErrClauseNil)
	}

	tokens := clause.RawTokens()
	meta := newWhereClauseMeta(clause.SourceText())

	var (
		clauseExprIndex    int
		hasClauseCondition bool
	)

	if cond := strings.TrimSpace(clause.IfCondition()); cond != "" {
		envIndex := builder.getCurrentEnvironmentIndex()
		clauseExprIndex = builder.context.AddExpression(cond, envIndex)

		if tokens := clause.RawTokens(); len(tokens) > 0 {
			start := tokens[0].Position
			builder.context.SetExpressionMetadata(clauseExprIndex, Position{Line: start.Line, Column: start.Column}, nil)
		}

		hasClauseCondition = true
		meta.Dynamic = true
		meta.ExpressionRefs = appendUniqueInt(meta.ExpressionRefs, clauseExprIndex)
		meta.DynamicConditions = append(meta.DynamicConditions, &WhereDynamicCondition{
			ExprIndex:        clauseExprIndex,
			NegatedWhenEmpty: true,
			Description:      cond,
		})
	}

	builder.pushWhereMeta(meta)
	defer builder.popWhereMeta()

	startIdx := builder.instructionCount()

	// Phase 1: トークンをそのまま処理
	// 将来的には、ここでトークンのカスタマイズを行う
	// 例: 条件式の最適化、サブクエリの処理など

	if err := builder.ProcessTokens(tokens); err != nil {
		return nil, fmt.Errorf("failed to process tokens in WHERE clause: %w", err)
	}

	meta.Finalize()

	fallbackInserted := false

	if critical && hasClauseCondition {
		pos := ""
		if len(tokens) > 0 {
			pos = tokens[0].Position.String()
		}

		clauseInstructions := append([]Instruction(nil), builder.instructions[startIdx:]...)
		builder.instructions = builder.instructions[:startIdx]
		exprCopy := clauseExprIndex
		builder.instructions = append(builder.instructions, Instruction{Op: OpIf, Pos: pos, ExprIndex: &exprCopy})
		builder.instructions = append(builder.instructions, clauseInstructions...)
		builder.instructions = append(builder.instructions, Instruction{Op: OpElse, Pos: pos})
		builder.instructions = append(builder.instructions, Instruction{
			Op:       OpFallbackCondition,
			Pos:      pos,
			Value:    "WHERE 1 = 1",
			Critical: critical,
		})
		builder.instructions = append(builder.instructions, Instruction{Op: OpEnd, Pos: pos})
		fallbackInserted = true
	}

	if hasClauseCondition {
		if len(meta.RemovalCombos) == 0 {
			meta.RemovalCombos = [][]RemovalLiteral{{{
				ExprIndex: clauseExprIndex,
				When:      false,
			}}}
		}

		meta.Status = StatusConditional
	}

	if !fallbackInserted && critical && meta.Status == StatusConditional && len(meta.RemovalCombos) > 0 {
		pos := ""
		if len(tokens) > 0 {
			pos = tokens[len(tokens)-1].Position.String()
		}

		builder.instructions = append(builder.instructions, Instruction{
			Op:             OpFallbackCondition,
			Pos:            pos,
			Value:          "1 = 1",
			Critical:       true,
			FallbackCombos: cloneRemovalCombos(meta.RemovalCombos),
		})
	}

	// WHERE句の終了時に BOUNDARY を追加
	// （ただし、末尾が END の場合は追加しない）
	builder.AddBoundary()

	return meta, nil
}

func cloneRemovalCombos(src [][]RemovalLiteral) [][]RemovalLiteral {
	if len(src) == 0 {
		return nil
	}

	out := make([][]RemovalLiteral, len(src))
	for i := range src {
		out[i] = append([]RemovalLiteral(nil), src[i]...)
	}

	return out
}
