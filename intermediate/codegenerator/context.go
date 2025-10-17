package codegenerator

import (
	"fmt"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
)

// GenerationContext は命令列生成時のコンテキスト情報を保持する
type GenerationContext struct {
	// CEL式のリスト（式のインデックス順）
	Expressions []CELExpression

	// CEL環境のリスト（ループ変数などの環境情報）
	CELEnvironments []CELEnvironment

	// 環境変数（ループ変数など）
	Environments []string

	// 方言設定（postgres, mysql, sqlite 等）
	Dialect snapsql.Dialect

	// テーブル情報（スキーマ情報）
	TableInfo map[string]*snapsql.TableInfo

	// パース済み AST（参照用、現在は未使用だが将来的に拡張可能）
	Statement parser.StatementNode
}

// NewGenerationContext creates a new GenerationContext with the root environment initialized.
// The root environment (index 0) is always created as the default environment for the query.
func NewGenerationContext(dialect snapsql.Dialect) *GenerationContext {
	ctx := &GenerationContext{
		Expressions:     make([]CELExpression, 0),
		CELEnvironments: make([]CELEnvironment, 0),
		Environments:    make([]string, 0),
		Dialect:         dialect,
		TableInfo:       nil,
		Statement:       nil,
	}

	// Initialize with root environment (index 0)
	rootEnv := CELEnvironment{
		Index:               0,
		AdditionalVariables: make([]CELVariableInfo, 0),
		Container:           "root",
		ParentIndex:         nil,
	}
	ctx.AddCELEnvironment(rootEnv)

	return ctx
}

// AddExpression adds a CEL expression to the context and returns its index.
// If the same expression already exists, returns the existing index.
func (ctx *GenerationContext) AddExpression(expr string, environmentIndex int) int {
	// Check if expression already exists
	for i, existingExpr := range ctx.Expressions {
		if existingExpr.Expression == expr && existingExpr.EnvironmentIndex == environmentIndex {
			return i
		}
	}
	// Add new expression
	index := len(ctx.Expressions)
	celExpr := CELExpression{
		ID:               fmt.Sprintf("expr_%03d", index+1),
		Expression:       expr,
		EnvironmentIndex: environmentIndex,
		Position: Position{
			Line:   0,
			Column: 0,
		},
	}
	ctx.Expressions = append(ctx.Expressions, celExpr)

	return index
}

// AddEnvironment adds an environment variable (loop variable) to the context and returns its index.
func (ctx *GenerationContext) AddEnvironment(env string) int {
	index := len(ctx.Environments)
	ctx.Environments = append(ctx.Environments, env)

	return index
}

// AddCELEnvironment adds a CEL environment with variable definitions and returns its index.
func (ctx *GenerationContext) AddCELEnvironment(env CELEnvironment) int {
	index := len(ctx.CELEnvironments)
	env.Index = index
	ctx.CELEnvironments = append(ctx.CELEnvironments, env)

	return index
}
