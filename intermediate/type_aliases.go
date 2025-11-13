package intermediate

import (
	"github.com/shibukawa/snapsql/explang"
	"github.com/shibukawa/snapsql/intermediate/codegenerator"
)

// Type aliases for backward compatibility
// これらの型は codegenerator パッケージに移動されましたが、
// 既存のコードとの互換性のためにエイリアスを提供します。

// Instruction is an alias for codegenerator.Instruction
type Instruction = codegenerator.Instruction

// CELExpression is an alias for codegenerator.CELExpression
type CELExpression = codegenerator.CELExpression

// CELEnvironment is an alias for codegenerator.CELEnvironment
type CELEnvironment = codegenerator.CELEnvironment

// CELVariableInfo is an alias for codegenerator.CELVariableInfo
type CELVariableInfo = codegenerator.CELVariableInfo

// Position is an alias for codegenerator.Position
type Position = codegenerator.Position

// Expressions is an alias for explang.Step
type Expressions = explang.Step

// ExpressionKind is an alias for explang.StepKind
type ExpressionKind = explang.StepKind

const (
	// StepIdentifier re-exports explang.StepIdentifier for intermediate consumers.
	StepIdentifier ExpressionKind = explang.StepIdentifier
	// StepMember re-exports explang.StepMember for intermediate consumers.
	StepMember ExpressionKind = explang.StepMember
	// StepIndex re-exports explang.StepIndex for intermediate consumers.
	StepIndex ExpressionKind = explang.StepIndex
)

// Op constants (re-exported from codegenerator)
const (
	OpEmitStatic         = codegenerator.OpEmitStatic
	OpEmitEval           = codegenerator.OpEmitEval
	OpEmitUnlessBoundary = codegenerator.OpEmitUnlessBoundary
	OpBoundary           = codegenerator.OpBoundary
	OpIf                 = codegenerator.OpIf
	OpElseIf             = codegenerator.OpElseIf
	OpElse               = codegenerator.OpElse
	OpEnd                = codegenerator.OpEnd
	OpLoopStart          = codegenerator.OpLoopStart
	OpLoopEnd            = codegenerator.OpLoopEnd
	OpIfSystemLimit      = codegenerator.OpIfSystemLimit
	OpIfSystemOffset     = codegenerator.OpIfSystemOffset
	OpEmitSystemLimit    = codegenerator.OpEmitSystemLimit
	OpEmitSystemOffset   = codegenerator.OpEmitSystemOffset
	OpEmitSystemValue    = codegenerator.OpEmitSystemValue
	OpEmitSystemFor      = codegenerator.OpEmitSystemFor
)
