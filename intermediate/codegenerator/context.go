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

	// SnapSQL設定（システムフィールドなど）
	Config *snapsql.Config

	// 関数定義（ダミー値生成用）
	FunctionDefinition *parser.FunctionDefinition

	// parserstep6 から受け取った型情報マップ（"line:col" -> descriptor）
	TypeInfoMap map[string]any
}

// NewGenerationContext creates a new GenerationContext with the root environment initialized.
// The root environment (index 0) is always created as the default environment for the query.
func NewGenerationContext(dialect snapsql.Dialect) *GenerationContext {
	ctx := &GenerationContext{
		Expressions:        make([]CELExpression, 0),
		CELEnvironments:    make([]CELEnvironment, 0),
		Environments:       make([]string, 0),
		Dialect:            dialect,
		TableInfo:          nil,
		Statement:          nil,
		Config:             nil,
		FunctionDefinition: nil,
		TypeInfoMap:        nil,
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

// SetExpressionMetadata updates position and type descriptor for the expression at index.
func (ctx *GenerationContext) SetExpressionMetadata(index int, pos Position, typeDescriptor any) {
	if index < 0 || index >= len(ctx.Expressions) {
		return
	}

	expr := &ctx.Expressions[index]

	extraPos := pos
	if extraPos.Line != 0 || extraPos.Column != 0 {
		expr.Position = extraPos
	}

	if typeDescriptor != nil {
		expr.TypeDescriptor = typeDescriptor
		expr.ResultType = DetermineEvalResultType(typeDescriptor)
	}
}

// GetExpressionType returns the classified result type of the expression.
func (ctx *GenerationContext) GetExpressionType(index int) EvalResultType {
	if index < 0 || index >= len(ctx.Expressions) {
		return EvalResultTypeUnknown
	}

	return ctx.Expressions[index].ResultType
}

// GetExpressionDescriptor returns the raw descriptor registered for the expression.
func (ctx *GenerationContext) GetExpressionDescriptor(index int) any {
	if index < 0 || index >= len(ctx.Expressions) {
		return nil
	}

	return ctx.Expressions[index].TypeDescriptor
}

// SetTypeInfoMap stores the parser-provided type information map for later lookups.
func (ctx *GenerationContext) SetTypeInfoMap(typeInfo map[string]any) {
	ctx.TypeInfoMap = typeInfo
}

// LookupTypeDescriptor returns the descriptor registered for a token位置 ("line:col").
func (ctx *GenerationContext) LookupTypeDescriptor(pos string) any {
	if ctx.TypeInfoMap == nil {
		return nil
	}

	if desc, ok := ctx.TypeInfoMap[pos]; ok {
		return desc
	}

	return nil
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

// SetConfig sets the SnapSQL configuration for this generation context.
// This is used to access system field definitions and other settings.
func (ctx *GenerationContext) SetConfig(config *snapsql.Config) {
	ctx.Config = config
}

// SetFunctionDefinition sets the function definition for this generation context.
// This is used to generate dummy values for CEL environment initialization.
func (ctx *GenerationContext) SetFunctionDefinition(funcDef *parser.FunctionDefinition) {
	ctx.FunctionDefinition = funcDef
}
