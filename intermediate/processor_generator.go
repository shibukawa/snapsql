package intermediate

import (
	"errors"
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/intermediate/codegenerator"
	"github.com/shibukawa/snapsql/parser"
)

// InstructionGenerator generates intermediate instructions using the unified codegenerator package.
var (
	errNilStatement             = errors.New("code generation requires a non-nil statement")
	errUnsupportedStatementType = errors.New("unsupported statement type")
)

type InstructionGenerator struct{}

func (i *InstructionGenerator) Name() string { return "InstructionGenerator" }

func (i *InstructionGenerator) Process(ctx *ProcessingContext) error {
	if ctx == nil || ctx.Statement == nil {
		return errNilStatement
	}

	genCtx := newGenerationContextFromProcessing(ctx)

	var (
		instructions []codegenerator.Instruction
		expressions  []codegenerator.CELExpression
		environments []codegenerator.CELEnvironment
		err          error
	)

	switch stmt := ctx.Statement.(type) {
	case *parser.SelectStatement:
		instructions, expressions, environments, err = codegenerator.GenerateSelectInstructions(stmt, genCtx)
	case *parser.InsertIntoStatement:
		instructions, expressions, environments, err = codegenerator.GenerateInsertInstructionsWithFunctionDef(stmt, genCtx, ctx.FunctionDef)
	case *parser.UpdateStatement:
		instructions, expressions, environments, err = codegenerator.GenerateUpdateInstructions(stmt, genCtx)
	case *parser.DeleteFromStatement:
		instructions, expressions, environments, err = codegenerator.GenerateDeleteInstructions(stmt, genCtx)
	default:
		return fmt.Errorf("%w: %T", errUnsupportedStatementType, ctx.Statement)
	}

	if err != nil {
		return fmt.Errorf("code generation failed: %w", err)
	}

	ctx.Instructions = instructions
	ctx.CELExpressions = expressions
	ctx.CELEnvironments = environments

	ctx.Environments = append([]string(nil), genCtx.Environments...)

	return nil
}

func newGenerationContextFromProcessing(ctx *ProcessingContext) *codegenerator.GenerationContext {
	dialect := resolveGenerationDialect(ctx.Dialect)
	genCtx := codegenerator.NewGenerationContext(dialect)
	genCtx.SetConfig(ctx.Config)
	genCtx.TableInfo = ctx.TableInfo
	genCtx.Statement = ctx.Statement
	genCtx.SetTypeInfoMap(ctx.TypeInfoMap)

	if ctx.FunctionDef != nil {
		genCtx.SetFunctionDefinition(ctx.FunctionDef)
	}

	return genCtx
}

func resolveGenerationDialect(dialect string) snapsql.Dialect {
	switch strings.ToLower(dialect) {
	case "mysql":
		return snapsql.DialectMySQL
	case "sqlite", "sqlite3":
		return snapsql.DialectSQLite
	case "mariadb":
		return snapsql.DialectMariaDB
	case "postgres", "postgresql", "pg":
		return snapsql.DialectPostgres
	default:
		return snapsql.DialectPostgres
	}
}
