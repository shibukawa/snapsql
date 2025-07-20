package parser

import (
	"errors"
	"fmt"
	"io"

	"github.com/shibukawa/snapsql/markdownparser"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
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
	StatementNode = cmn.StatementNode
	ClauseNode    = cmn.ClauseNode
	AstNode       = cmn.AstNode

	// Statement types
	SelectStatement     = cmn.SelectStatement
	InsertIntoStatement = cmn.InsertIntoStatement
	UpdateStatement     = cmn.UpdateStatement
	DeleteFromStatement = cmn.DeleteFromStatement

	// Clause types
	SelectClause     = cmn.SelectClause
	FromClause       = cmn.FromClause
	WhereClause      = cmn.WhereClause
	GroupByClause    = cmn.GroupByClause
	HavingClause     = cmn.HavingClause
	OrderByClause    = cmn.OrderByClause
	LimitClause      = cmn.LimitClause
	OffsetClause     = cmn.OffsetClause
	WithClause       = cmn.WithClause
	ForClause        = cmn.ForClause
	InsertIntoClause = cmn.InsertIntoClause
	ValuesClause     = cmn.ValuesClause
	UpdateClause     = cmn.UpdateClause
	SetClause        = cmn.SetClause
	DeleteFromClause = cmn.DeleteFromClause
	OnConflictClause = cmn.OnConflictClause
	ReturningClause  = cmn.ReturningClause

	// Element types
	FieldName   = cmn.FieldName
	FieldType   = cmn.FieldType
	SelectField = cmn.SelectField

	// Schema and namespace types
	FunctionDefinition = cmn.FunctionDefinition
	Namespace          = cmn.Namespace

	// Error types
	ParseError = cmn.ParseError

	// Subquery analysis types
	SubqueryAnalysisResult = cmn.SubqueryAnalysisResult
	ValidationError        = cmn.ValidationError
	SQDependencyGraph      = cmn.SQDependencyGraph
	SQFieldSource          = cmn.SQFieldSource
	SQTableReference       = cmn.SQTableReference
	SQDependencyNode       = cmn.SQDependencyNode
	SQScopeManager         = cmn.SQScopeManager
	SQDependencyType       = cmn.SQDependencyType
	SQSourceType           = cmn.SQSourceType
	SQErrorType            = cmn.SQErrorType
	SQParseError           = cmn.SQParseError

	// Node type constants
	NodeType = cmn.NodeType
)

// Re-export constants
const (
	// SQL statement structures
	UNKNOWN            = cmn.UNKNOWN
	SUBQUERY_STATEMENT = cmn.SUBQUERY_STATEMENT

	// Select statement
	SELECT_STATEMENT = cmn.SELECT_STATEMENT
	SELECT_CLAUSE    = cmn.SELECT_CLAUSE
	FROM_CLAUSE      = cmn.FROM_CLAUSE
	WHERE_CLAUSE     = cmn.WHERE_CLAUSE
	ORDER_BY_CLAUSE  = cmn.ORDER_BY_CLAUSE
	GROUP_BY_CLAUSE  = cmn.GROUP_BY_CLAUSE
	HAVING_CLAUSE    = cmn.HAVING_CLAUSE
	LIMIT_CLAUSE     = cmn.LIMIT_CLAUSE
	OFFSET_CLAUSE    = cmn.OFFSET_CLAUSE
	WITH_CLAUSE      = cmn.WITH_CLAUSE
	FOR_CLAUSE       = cmn.FOR_CLAUSE
	CTE_DEFINITION   = cmn.CTE_DEFINITION

	// Insert statement
	INSERT_INTO_STATEMENT = cmn.INSERT_INTO_STATEMENT
	INSERT_INTO_CLAUSE    = cmn.INSERT_INTO_CLAUSE
	VALUES_CLAUSE         = cmn.VALUES_CLAUSE
	ON_CONFLICT_CLAUSE    = cmn.ON_CONFLICT_CLAUSE

	// Update statement
	UPDATE_STATEMENT = cmn.UPDATE_STATEMENT
	UPDATE_CLAUSE    = cmn.UPDATE_CLAUSE
	SET_CLAUSE       = cmn.SET_CLAUSE

	// Delete statement
	DELETE_FROM_CLAUSE    = cmn.DELETE_FROM_CLAUSE
	DELETE_FROM_STATEMENT = cmn.DELETE_FROM_STATEMENT

	// FieldType constants
	SingleField   = cmn.SingleField
	TableField    = cmn.TableField
	FunctionField = cmn.FunctionField
	ComplexField  = cmn.ComplexField
	LiteralField  = cmn.LiteralField

	// parserstep7 dependency type constants
	DependencyCTE            = cmn.SQDependencyCTE
	DependencySubquery       = cmn.SQDependencySubquery
	DependencyMain           = cmn.SQDependencyMain
	DependencyFromSubquery   = cmn.SQDependencyFromSubquery
	DependencySelectSubquery = cmn.SQDependencySelectSubquery
)

// Re-export sentinel errors
var (
	// Parser related errors
	ErrInvalidSQL        = cmn.ErrInvalidSQL
	ErrInvalidForSnapSQL = cmn.ErrInvalidForSnapSQL

	// YAML/Schema related errors
	ErrExpectedDocumentNode     = cmn.ErrExpectedDocumentNode
	ErrExpectedMappingNode      = cmn.ErrExpectedMappingNode
	ErrExpectedMappingForParams = cmn.ErrExpectedMappingForParams
	ErrExpectedSequenceNode     = cmn.ErrExpectedSequenceNode
	ErrUnsupportedParameterType = cmn.ErrUnsupportedParameterType

	// CEL related errors
	ErrEnvironmentCELNotInit      = cmn.ErrEnvironmentCELNotInit
	ErrParameterCELNotInit        = cmn.ErrParameterCELNotInit
	ErrNoOutputType               = cmn.ErrNoOutputType
	ErrExpressionValidationFailed = cmn.ErrExpressionValidationFailed
	ErrExpressionNotList          = cmn.ErrExpressionNotList

	// Other errors
	ErrParameterNotFound = cmn.ErrParameterNotFound
)

// Re-export helper functions
var (
	NewNamespace = cmn.NewNamespace
	AsParseError = cmn.AsParseError
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

// RawParse is the main entry point for parsing SQL templates from pre-tokenized tokens.
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
func RawParse(tokens []tokenizer.Token, functionDef *FunctionDefinition, options *ParseOptions) (StatementNode, error) {
	if options == nil {
		options = &ParseOptions{}
	}

	// Step 1: Run parserstep1 - Basic syntax validation and dummy literal insertion
	processedTokens, err := parserstep1.Execute(tokens)
	if err != nil {
		return nil, fmt.Errorf("parserstep1 failed: %w", err)
	}

	// Step 2: Run parserstep2 - SQL structure parsing
	stmt, err := parserstep2.Execute(processedTokens)
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

	// Use ExecuteWithFunctionDef only if functionDef is provided
	if functionDef != nil {
		if parseErr := parserstep6.ExecuteWithFunctionDef(stmt, namespace, *functionDef); parseErr != nil {
			return nil, fmt.Errorf("parserstep6 failed: %w", parseErr)
		}
	} else {
		if parseErr := parserstep6.Execute(stmt, namespace); parseErr != nil {
			return nil, fmt.Errorf("parserstep6 failed: %w", parseErr)
		}
	}

	// Step 7: Run parserstep7 - Subquery dependency analysis (optional)
	if options.EnableSubqueryAnalysis {
		subqueryParser := parserstep7.NewSubqueryParserIntegrated()
		subErr := subqueryParser.ParseStatement(stmt, functionDef)

		if subErr != nil {
			// Don't fail the entire parse for subquery analysis errors
			// The error can be detected via stmt.GetSubqueryDependencies() == nil
			// This allows graceful degradation
		}
	}

	return stmt, nil
}

// ParseSQLFile parses an SQL file from an io.Reader and returns a StatementNode.
// This is a convenience function that handles tokenization internally.
//
// Parameters:
//   - reader: An io.Reader containing the SQL content
//   - functionDef: Function definition schema (optional, can be nil)
//   - options: Optional parsing options for environment and parameter values
//
// Returns:
//   - StatementNode: The parsed statement AST
//   - error: Any parsing errors encountered
func ParseSQLFile(reader io.Reader, functionDef *FunctionDefinition, options *ParseOptions) (StatementNode, error) {
	// Read all content from the reader
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read SQL content: %w", err)
	}

	// Create a tokenizer
	sqlTokenizer := tokenizer.NewSqlTokenizer(string(content))

	// Collect all tokens
	var tokens []tokenizer.Token
	for token, err := range sqlTokenizer.Tokens() {
		if err != nil {
			return nil, fmt.Errorf("tokenization failed: %w", err)
		}
		tokens = append(tokens, token)
	}

	// Parse the tokens
	return RawParse(tokens, functionDef, options)
}

// ParseMarkdownFile parses a SnapSQLDocument and returns a StatementNode.
// This function extracts the SQL and parameters from the document and parses them.
//
// Parameters:
//   - doc: A SnapSQLDocument from the markdownparser package
//   - basePath: Base path for resolving relative paths in common type references
//   - projectRootPath: Project root path for resolving common type references
//   - options: Optional parsing options for environment and parameter values
//
// Returns:
//   - StatementNode: The parsed statement AST
//   - error: Any parsing errors encountered
func ParseMarkdownFile(doc *markdownparser.SnapSQLDocument, basePath string, projectRootPath string, options *ParseOptions) (StatementNode, error) {
	// Create a function definition from the SnapSQLDocument
	functionDef, err := cmn.ParseFunctionDefinitionFromSnapSQLDocument(doc, basePath, projectRootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create function definition: %w", err)
	}

	// Create a tokenizer for the SQL content
	sqlTokenizer := tokenizer.NewSqlTokenizer(doc.SQL)

	// Collect all tokens
	var tokens []tokenizer.Token
	for token, err := range sqlTokenizer.Tokens() {
		if err != nil {
			return nil, fmt.Errorf("tokenization failed: %w", err)
		}
		tokens = append(tokens, token)
	}

	// Parse the tokens
	return RawParse(tokens, functionDef, options)
}
