package parser

import (
	"errors"
	"fmt"

	"github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/parser/parserstep1"
	"github.com/shibukawa/snapsql/parser/parserstep2"
	"github.com/shibukawa/snapsql/parser/parserstep3"
	"github.com/shibukawa/snapsql/parser/parserstep4"
	"github.com/shibukawa/snapsql/parser/parserstep5"
	"github.com/shibukawa/snapsql/parser/parserstep6"
	"github.com/shibukawa/snapsql/parser/parserstep7"
	"github.com/shibukawa/snapsql/tokenizer"
)

// Sentinel errors for parser operations
var (
	ErrSubqueryAnalysisNotAvailable = errors.New("subquery analysis not available")
)

// Re-export common types for user convenience
type (
	// Core interfaces
	StatementNode = parsercommon.StatementNode
	ClauseNode    = parsercommon.ClauseNode
	AstNode       = parsercommon.AstNode

	// Statement types
	SelectStatement     = parsercommon.SelectStatement
	InsertIntoStatement = parsercommon.InsertIntoStatement
	UpdateStatement     = parsercommon.UpdateStatement
	DeleteFromStatement = parsercommon.DeleteFromStatement

	// Clause types
	SelectClause     = parsercommon.SelectClause
	FromClause       = parsercommon.FromClause
	WhereClause      = parsercommon.WhereClause
	GroupByClause    = parsercommon.GroupByClause
	HavingClause     = parsercommon.HavingClause
	OrderByClause    = parsercommon.OrderByClause
	LimitClause      = parsercommon.LimitClause
	OffsetClause     = parsercommon.OffsetClause
	WithClause       = parsercommon.WithClause
	ForClause        = parsercommon.ForClause
	InsertIntoClause = parsercommon.InsertIntoClause
	ValuesClause     = parsercommon.ValuesClause
	UpdateClause     = parsercommon.UpdateClause
	SetClause        = parsercommon.SetClause
	DeleteFromClause = parsercommon.DeleteFromClause
	OnConflictClause = parsercommon.OnConflictClause
	ReturningClause  = parsercommon.ReturningClause

	// Element types
	FieldName = parsercommon.FieldName
	FieldType = parsercommon.FieldType

	// Schema and namespace types
	FunctionDefinition = parsercommon.FunctionDefinition
	Namespace          = parsercommon.Namespace
	CELVariable        = parsercommon.CELVariable

	// Error types
	ParseError = parsercommon.ParseError

	// Node type constants
	NodeType = parsercommon.NodeType
)

// Re-export constants
const (
	// SQL statement structures
	UNKNOWN            = parsercommon.UNKNOWN
	SUBQUERY_STATEMENT = parsercommon.SUBQUERY_STATEMENT

	// Select statement
	SELECT_STATEMENT = parsercommon.SELECT_STATEMENT
	SELECT_CLAUSE    = parsercommon.SELECT_CLAUSE
	FROM_CLAUSE      = parsercommon.FROM_CLAUSE
	WHERE_CLAUSE     = parsercommon.WHERE_CLAUSE
	ORDER_BY_CLAUSE  = parsercommon.ORDER_BY_CLAUSE
	GROUP_BY_CLAUSE  = parsercommon.GROUP_BY_CLAUSE
	HAVING_CLAUSE    = parsercommon.HAVING_CLAUSE
	LIMIT_CLAUSE     = parsercommon.LIMIT_CLAUSE
	OFFSET_CLAUSE    = parsercommon.OFFSET_CLAUSE
	WITH_CLAUSE      = parsercommon.WITH_CLAUSE
	FOR_CLAUSE       = parsercommon.FOR_CLAUSE
	CTE_DEFINITION   = parsercommon.CTE_DEFINITION

	// Insert statement
	INSERT_INTO_STATEMENT = parsercommon.INSERT_INTO_STATEMENT
	INSERT_INTO_CLAUSE    = parsercommon.INSERT_INTO_CLAUSE
	VALUES_CLAUSE         = parsercommon.VALUES_CLAUSE
	ON_CONFLICT_CLAUSE    = parsercommon.ON_CONFLICT_CLAUSE

	// Update statement
	UPDATE_STATEMENT = parsercommon.UPDATE_STATEMENT
	UPDATE_CLAUSE    = parsercommon.UPDATE_CLAUSE
	SET_CLAUSE       = parsercommon.SET_CLAUSE

	// Delete statement
	DELETE_FROM_CLAUSE    = parsercommon.DELETE_FROM_CLAUSE
	DELETE_FROM_STATEMENT = parsercommon.DELETE_FROM_STATEMENT

	// FieldType constants
	SingleField   = parsercommon.SingleField
	TableField    = parsercommon.TableField
	FunctionField = parsercommon.FunctionField
	ComplexField  = parsercommon.ComplexField
	LiteralField  = parsercommon.LiteralField
)

// Re-export sentinel errors
var (
	// Parser related errors
	ErrInvalidSQL        = parsercommon.ErrInvalidSQL
	ErrInvalidForSnapSQL = parsercommon.ErrInvalidForSnapSQL

	// YAML/Schema related errors
	ErrExpectedDocumentNode     = parsercommon.ErrExpectedDocumentNode
	ErrExpectedMappingNode      = parsercommon.ErrExpectedMappingNode
	ErrExpectedMappingForParams = parsercommon.ErrExpectedMappingForParams
	ErrExpectedSequenceNode     = parsercommon.ErrExpectedSequenceNode
	ErrUnsupportedParameterType = parsercommon.ErrUnsupportedParameterType

	// CEL related errors
	ErrEnvironmentCELNotInit      = parsercommon.ErrEnvironmentCELNotInit
	ErrParameterCELNotInit        = parsercommon.ErrParameterCELNotInit
	ErrNoOutputType               = parsercommon.ErrNoOutputType
	ErrExpressionValidationFailed = parsercommon.ErrExpressionValidationFailed
	ErrExpressionNotList          = parsercommon.ErrExpressionNotList

	// Other errors
	ErrParameterNotFound = parsercommon.ErrParameterNotFound
)

// Re-export helper functions
var (
	NewNamespace = parsercommon.NewNamespace
	AsParseError = parsercommon.AsParseError
)

// ParseOptions contains options for the Parse function
type ParseOptions struct {
	// Environment variables for CEL evaluation
	Environment map[string]any
	// Parameter values for CEL evaluation (optional, will generate dummy data if nil)
	Parameters map[string]any
	// Enable subquery dependency analysis (parserstep7)
	EnableSubqueryAnalysis bool
}

// Parse is the main entry point for parsing SQL templates from pre-tokenized tokens.
// It takes tokenized SQL and optional additional YAML function definitions,
// runs the complete parsing pipeline (parserstep1-6), and returns a StatementNode.
//
// When EnableSubqueryAnalysis is true, parserstep7 will be executed and its results
// will be stored directly in the StatementNode for easy access via:
// - stmt.GetFieldSources()
// - stmt.GetTableReferences()
// - stmt.GetSubqueryDependencies()
//
// Parameters:
//   - tokens: Pre-tokenized SQL tokens
//   - functionDef: Function definition schema
//   - options: Optional parsing options for environment and parameter values
//
// Returns:
//   - StatementNode: The parsed statement AST (may contain parserstep7 results)
//   - error: Any parsing errors encountered
func Parse(tokens []tokenizer.Token, functionDef *FunctionDefinition, options *ParseOptions) (StatementNode, error) {
	if options == nil {
		options = &ParseOptions{}
	}

	// Step 1: Run parserstep1 - Basic syntax validation
	if err := parserstep1.Execute(tokens); err != nil {
		return nil, fmt.Errorf("parserstep1 failed: %w", err)
	}

	// Step 2: Run parserstep2 - SQL structure parsing
	stmt, err := parserstep2.Execute(tokens)
	if err != nil {
		return nil, fmt.Errorf("parserstep2 failed: %w", err)
	}

	// Step 3: Run parserstep3 - Clause-level validation and assignment
	if err := parserstep3.Execute(stmt); err != nil {
		return nil, fmt.Errorf("parserstep3 failed: %w", err)
	}

	// Step 4: Run parserstep4 - Clause content validation
	if err := parserstep4.Execute(stmt); err != nil {
		return nil, fmt.Errorf("parserstep4 failed: %w", err)
	}

	// Step 5: Run parserstep5 - Directive structure validation
	if err := parserstep5.Execute(stmt); err != nil {
		return nil, fmt.Errorf("parserstep5 failed: %w", err)
	}

	// Step 6: Run parserstep6 - Variable and directive validation
	// Create namespace from function definition if available
	var environment map[string]any
	var parameters map[string]any

	if options.Environment != nil {
		environment = options.Environment
	} else {
		environment = make(map[string]any)
	}

	if options.Parameters != nil {
		parameters = options.Parameters
	}

	namespace := NewNamespace(functionDef, environment, parameters)

	if parseErr := parserstep6.Execute(stmt, namespace); parseErr != nil {
		return nil, fmt.Errorf("parserstep6 failed: %w", parseErr)
	}

	// Step 7: Run parserstep7 - Subquery dependency analysis (optional)
	if options.EnableSubqueryAnalysis {
		subqueryParser := parserstep7.NewSubqueryParserIntegrated()
		_, subErr := subqueryParser.ParseStatement(stmt)

		if subErr != nil {
			// Don't fail the entire parse for subquery analysis errors
			// The error can be detected via stmt.GetSubqueryDependencies() == nil
			// This allows graceful degradation
		}
	}

	return stmt, nil
}
