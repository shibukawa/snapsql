package parser

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

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
	// StatementNode is the core interface for all statement-level AST nodes (re-export).
	// Core interfaces
	StatementNode = cmn.StatementNode
	// ClauseNode represents a clause-level AST node (re-export).
	ClauseNode    = cmn.ClauseNode
	// AstNode is the base interface for every parsed SQL AST node (re-export).
	AstNode       = cmn.AstNode

	// SelectStatement represents a parsed SELECT statement (re-export).
	SelectStatement     = cmn.SelectStatement
	// InsertIntoStatement represents a parsed INSERT ... INTO statement (re-export).
	InsertIntoStatement = cmn.InsertIntoStatement
	// UpdateStatement represents a parsed UPDATE statement (re-export).
	UpdateStatement     = cmn.UpdateStatement
	// DeleteFromStatement represents a parsed DELETE FROM statement (re-export).
	DeleteFromStatement = cmn.DeleteFromStatement

	// SelectClause represents the SELECT clause (re-export).
	SelectClause     = cmn.SelectClause
	// FromClause represents the FROM clause (re-export).
	FromClause       = cmn.FromClause
	// WhereClause represents the WHERE clause (re-export).
	WhereClause      = cmn.WhereClause
	// GroupByClause represents the GROUP BY clause (re-export).
	GroupByClause    = cmn.GroupByClause
	// HavingClause represents the HAVING clause (re-export).
	HavingClause     = cmn.HavingClause
	// OrderByClause represents the ORDER BY clause (re-export).
	OrderByClause    = cmn.OrderByClause
	// LimitClause represents the LIMIT clause (re-export).
	LimitClause      = cmn.LimitClause
	// OffsetClause represents the OFFSET clause (re-export).
	OffsetClause     = cmn.OffsetClause
	// WithClause represents the WITH clause (re-export).
	WithClause       = cmn.WithClause
	// ForClause represents the FOR clause (re-export).
	ForClause        = cmn.ForClause
	// InsertIntoClause represents the INSERT INTO clause (re-export).
	InsertIntoClause = cmn.InsertIntoClause
	// ValuesClause represents the VALUES clause (re-export).
	ValuesClause     = cmn.ValuesClause
	// UpdateClause represents the UPDATE clause (re-export).
	UpdateClause     = cmn.UpdateClause
	// SetClause represents the SET clause (re-export).
	SetClause        = cmn.SetClause
	// DeleteFromClause represents the DELETE FROM clause (re-export).
	DeleteFromClause = cmn.DeleteFromClause
	// OnConflictClause represents the ON CONFLICT clause (re-export).
	OnConflictClause = cmn.OnConflictClause
	// ReturningClause represents the RETURNING clause (re-export).
	ReturningClause  = cmn.ReturningClause

	// FieldName represents a bare field name (re-export).
	FieldName   = cmn.FieldName
	// FieldType represents a categorized field type (re-export).
	FieldType   = cmn.FieldType
	// SelectField represents one field expression in a SELECT list (re-export).
	SelectField = cmn.SelectField

	// FunctionDefinition represents a function signature definition (re-export).
	FunctionDefinition = cmn.FunctionDefinition
	// Namespace represents a logical namespace grouping (re-export).
	Namespace          = cmn.Namespace
	// SetAssign represents a SET assignment expression (re-export).
	SetAssign          = cmn.SetAssign

	// ParseError represents a parsing error (re-export).
	ParseError = cmn.ParseError

	// SubqueryAnalysisResult contains analysis metadata for subqueries (re-export).
	SubqueryAnalysisResult = cmn.SubqueryAnalysisResult
	// ValidationError represents a validation error (re-export).
	ValidationError        = cmn.ValidationError
	// SQDependencyGraph represents the subquery dependency graph (re-export).
	SQDependencyGraph      = cmn.SQDependencyGraph
	// SQFieldSource represents a field source in dependency analysis (re-export).
	SQFieldSource          = cmn.SQFieldSource
	// SQTableReference represents a table reference (re-export).
	SQTableReference       = cmn.SQTableReference
	// SQDependencyNode represents a node in dependency graph (re-export).
	SQDependencyNode       = cmn.SQDependencyNode
	// SQScopeManager manages scopes for dependency analysis (re-export).
	SQScopeManager         = cmn.SQScopeManager
	// SQDependencyType enumerates dependency relationship types (re-export).
	SQDependencyType       = cmn.SQDependencyType
	// SQSourceType enumerates source types for dependency nodes (re-export).
	SQSourceType           = cmn.SQSourceType
	// SQErrorType enumerates error categories used in parsing (re-export).
	SQErrorType            = cmn.SQErrorType
	// SQParseError represents a parse error enriched with context (re-export).
	SQParseError           = cmn.SQParseError

	// NodeType is the re-export of cmn.NodeType.
	NodeType = cmn.NodeType
	// JoinType is the re-export of cmn.JoinType.
	JoinType = cmn.JoinType
)

// Re-export constants
const (
	// UNKNOWN indicates unknown node type (re-export).
	// SQL statement structures
	UNKNOWN            = cmn.UNKNOWN
	SUBQUERY_STATEMENT = cmn.SUBQUERY_STATEMENT

	// SELECT_STATEMENT indicates a SELECT statement node type (re-export).
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

	// INSERT_INTO_STATEMENT represents an INSERT statement node type.
	INSERT_INTO_STATEMENT = cmn.INSERT_INTO_STATEMENT
	INSERT_INTO_CLAUSE    = cmn.INSERT_INTO_CLAUSE
	VALUES_CLAUSE         = cmn.VALUES_CLAUSE
	ON_CONFLICT_CLAUSE    = cmn.ON_CONFLICT_CLAUSE

	// UPDATE_STATEMENT represents an UPDATE statement node type.
	UPDATE_STATEMENT = cmn.UPDATE_STATEMENT
	UPDATE_CLAUSE    = cmn.UPDATE_CLAUSE
	SET_CLAUSE       = cmn.SET_CLAUSE

	// DELETE_FROM_CLAUSE represents a DELETE FROM clause node type.
	DELETE_FROM_CLAUSE    = cmn.DELETE_FROM_CLAUSE
	// DELETE_FROM_STATEMENT represents a full DELETE statement node type.
	DELETE_FROM_STATEMENT = cmn.DELETE_FROM_STATEMENT

	// SingleField is a FieldType for single unqualified field names.
	SingleField   = cmn.SingleField
	// TableField is a FieldType for qualified table.field names.
	TableField = cmn.TableField
	// FunctionField is a FieldType for function call expressions.
	FunctionField = cmn.FunctionField
	// ComplexField is a FieldType for complex expressions.
	ComplexField = cmn.ComplexField
	// LiteralField is a FieldType for literal values.
	LiteralField = cmn.LiteralField

	// DependencyCTE represents a dependency on a CTE.
	DependencyCTE            = cmn.SQDependencyCTE
	// DependencySubquery represents a generic subquery dependency.
	DependencySubquery = cmn.SQDependencySubquery
	// DependencyMain represents the main query dependency.
	DependencyMain = cmn.SQDependencyMain
	// DependencyFromSubquery represents a FROM subquery dependency.
	DependencyFromSubquery = cmn.SQDependencyFromSubquery
	// DependencySelectSubquery represents a SELECT list subquery dependency.
	DependencySelectSubquery = cmn.SQDependencySelectSubquery

	JoinNone         = cmn.JoinNone
	JoinInner        = cmn.JoinInner
	JoinLeft         = cmn.JoinLeft
	JoinRight        = cmn.JoinRight
	JoinFull         = cmn.JoinFull
	JoinCross        = cmn.JoinCross
	JoinNatural      = cmn.JoinNatural
	JoinNaturalLeft  = cmn.JoinNaturalLeft
	JoinNaturalRight = cmn.JoinNaturalRight
	JoinNaturalFull  = cmn.JoinNaturalFull
)

// Re-export sentinel errors
var (
	// ErrInvalidSQL indicates the SQL was syntactically invalid.
	ErrInvalidSQL = cmn.ErrInvalidSQL
	// ErrInvalidForSnapSQL indicates the SQL uses features unsupported by SnapSQL.
	ErrInvalidForSnapSQL = cmn.ErrInvalidForSnapSQL

	// ErrExpectedDocumentNode indicates a YAML root document node was expected.
	ErrExpectedDocumentNode = cmn.ErrExpectedDocumentNode
	// ErrExpectedMappingForParams indicates a mapping node was expected for parameters.
	ErrExpectedMappingForParams = cmn.ErrExpectedMappingForParams
	// ErrExpectedSequenceNode indicates a sequence node was expected.
	ErrExpectedSequenceNode = cmn.ErrExpectedSequenceNode
	// ErrUnsupportedParameterType indicates a parameter type is unsupported.
	ErrUnsupportedParameterType = cmn.ErrUnsupportedParameterType

	// ErrEnvironmentCELNotInit indicates CEL environment wasn't initialized.
	ErrEnvironmentCELNotInit = cmn.ErrEnvironmentCELNotInit
	// ErrParameterCELNotInit indicates CEL parameter environment wasn't initialized.
	ErrParameterCELNotInit = cmn.ErrParameterCELNotInit
	// ErrNoOutputType indicates no output type was specified.
	ErrNoOutputType = cmn.ErrNoOutputType
	// ErrExpressionValidationFailed indicates expression failed validation.
	ErrExpressionValidationFailed = cmn.ErrExpressionValidationFailed
	// ErrExpressionNotList indicates expression result wasn't a list.
	ErrExpressionNotList = cmn.ErrExpressionNotList

	// ErrParameterNotFound indicates a referenced parameter was not found.
	ErrParameterNotFound = cmn.ErrParameterNotFound
)

// Re-export helper functions
var (
	AsParseError = cmn.AsParseError
)

// RawParse is the main entry point for parsing SQL templates from pre-tokenized tokens.
// It takes tokenized SQL and function definition, runs the complete parsing pipeline (parserstep1-7),
// and returns a StatementNode.
//
// Parameters:
//   - tokens: Pre-tokenized SQL tokens
//   - functionDef: Function definition schema (always required)
//   - constants: Optional constants for CEL evaluation
//
// Returns:
//   - StatementNode: The parsed statement AST with subquery analysis results
//   - error: Any parsing errors encountered
//
// RawParse is the main entry point for parsing SQL templates from pre-tokenized tokens.
// It runs the complete parsing pipeline with the given options.
func RawParse(tokens []tokenizer.Token, functionDef *FunctionDefinition, constants map[string]any, opts Options) (StatementNode, error) {
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
	err = parserstep3.Execute(stmt)
	if err != nil {
		return nil, fmt.Errorf("parserstep3 failed: %w", err)
	}

	// Step 4: Run parserstep4 - Clause content validation
	// Use InspectMode to relax certain validations (e.g., NATURAL JOIN, asterisk)
	err = parserstep4.ExecuteWithOptions(stmt, opts.InspectMode)
	if err != nil {
		return nil, fmt.Errorf("parserstep4 failed: %w", err)
	}

	// Step 5: Run parserstep5 - Directive structure validation
	// Use relaxed behavior in InspectMode
	err = parserstep5.ExecuteWithOptions(stmt, functionDef, opts.InspectMode)
	if err != nil {
		return nil, fmt.Errorf("parserstep5 failed: %w", err)
	}

	// Step 6: Run parserstep6 - Variable and directive validation
	// Create namespace from function definition for parameters
	paramNamespace, err := cmn.NewNamespaceFromDefinition(functionDef)
	if err != nil {
		return nil, fmt.Errorf("failed to create parameter namespace: %w", err)
	}

	// Create a separate namespace for constants if provided
	var constNamespace *cmn.Namespace
	if len(constants) > 0 {
		constNamespace, err = cmn.NewNamespaceFromConstants(constants)
		if err != nil {
			return nil, fmt.Errorf("failed to create constants namespace: %w", err)
		}
	} else {
		// Create an empty constants namespace if none provided
		constNamespace, err = cmn.NewNamespaceFromConstants(make(map[string]any))
		if err != nil {
			return nil, fmt.Errorf("failed to create empty constants namespace: %w", err)
		}
	}

	// Execute parserstep6 with both namespaces
	parseErr := parserstep6.ExecuteWithOptions(stmt, paramNamespace, constNamespace, opts.InspectMode)
	if parseErr != nil {
		return nil, fmt.Errorf("parserstep6 failed: %w", parseErr)
	}

	// Step 7: Run parserstep7 - Subquery dependency analysis (always enabled)
	subqueryParser := parserstep7.NewSubqueryParserIntegrated()

	subErr := subqueryParser.ParseStatement(stmt, functionDef)
	if subErr != nil {
		// Don't fail the entire parse for subquery analysis errors
		// The error can be detected via stmt.GetSubqueryDependencies() == nil
		// This allows graceful degradation
		_ = subErr // Explicitly ignore the error for now
	}

	return stmt, nil
}

// ParseSQLFile parses an SQL file from an io.Reader and returns a StatementNode and FunctionDefinition.
// This is a convenience function that handles tokenization internally.
// It always attempts to extract the function definition from SQL comments.
//
// Parameters:
//   - reader: An io.Reader containing the SQL content
//   - constants: Optional constants for CEL evaluation
//   - basePath: Base path for resolving relative paths in common type references (optional)
//   - projectRootPath: Project root path for resolving common type references (optional)
//
// Returns:
//   - StatementNode: The parsed statement AST
//   - *FunctionDefinition: The function definition extracted from the SQL file
//   - error: Any parsing errors encountered
//
// ParseSQLFile parses an SQL file from an io.Reader with options.
func ParseSQLFile(reader io.Reader, constants map[string]any, basePath string, projectRootPath string, opts Options) (StatementNode, *FunctionDefinition, error) {
	// Read all content from the reader
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read SQL content: %w", err)
	}

	// Tokenize the SQL content
	tokens, err := tokenizer.Tokenize(string(content))
	if err != nil {
		return nil, nil, fmt.Errorf("tokenization failed: %w", err)
	}

	// Extract function definition from SQL comments (fallback to file name if absent)
	functionDef, err := cmn.ParseFunctionDefinitionFromSQLComment(tokens, basePath, projectRootPath)
	if err != nil {
		if cmn.IsNoFunctionDefinition(err) {
			def := &cmn.FunctionDefinition{}

			if basePath != "" {
				base := filepath.Base(basePath)

				var name string
				// normalize extension comparison
				lower := strings.ToLower(base)
				if strings.HasSuffix(lower, ".snap.sql") {
					name = base[:len(base)-len(".snap.sql")]
				} else {
					name = strings.TrimSuffix(base, filepath.Ext(base))
				}

				if name != "" {
					def.FunctionName = name
				}
			}

			if err := def.Finalize(basePath, projectRootPath); err != nil {
				return nil, nil, fmt.Errorf("failed to finalize fallback function definition: %w", err)
			}

			functionDef = def
		} else {
			return nil, nil, err
		}
	}

	stmt, err := RawParse(tokens, functionDef, constants, opts)

	return stmt, functionDef, err
}

// ParseMarkdownFile parses a SnapSQLDocument and returns a StatementNode and FunctionDefinition.
// This function extracts the SQL and parameters from the document and parses them.
//
// Parameters:
//   - doc: A SnapSQLDocument from the markdownparser package
//   - basePath: Base path for resolving relative paths in common type references
//   - projectRootPath: Project root path for resolving common type references
//   - constants: Optional constants for CEL evaluation
//
// Returns:
//   - StatementNode: The parsed statement AST
//   - *FunctionDefinition: The function definition extracted from the document
//   - error: Any parsing errors encountered
//
// ParseMarkdownFile parses a SnapSQLDocument with options.
func ParseMarkdownFile(doc *markdownparser.SnapSQLDocument, basePath string, projectRootPath string, constants map[string]any, opts Options) (StatementNode, *FunctionDefinition, error) {
	// Create a function definition from the SnapSQLDocument
	functionDef, err := cmn.ParseFunctionDefinitionFromSnapSQLDocument(doc, basePath, projectRootPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create function definition: %w", err)
	}

	// Finalize the function definition to generate dummy data
	if err := functionDef.Finalize(basePath, projectRootPath); err != nil {
		return nil, functionDef, fmt.Errorf("failed to finalize function definition: %w", err)
	}

	// Merge constants with dummy data from function definition
	mergedConstants := make(map[string]any)

	for k, v := range constants {
		mergedConstants[k] = v
	}
	// Add dummy data (dummy data takes precedence if constants is nil or doesn't contain the key)
	if dummyDataAny := functionDef.DummyData(); dummyDataAny != nil {
		if dummyData, ok := dummyDataAny.(map[string]any); ok {
			for k, v := range dummyData {
				if _, exists := mergedConstants[k]; !exists {
					mergedConstants[k] = v
				}
			}
		}
	}

	// Tokenize the SQL content with line offset from markdown
	tokens, err := tokenizer.Tokenize(doc.SQL, doc.SQLStartLine)
	if err != nil {
		return nil, functionDef, fmt.Errorf("tokenization failed: %w", err)
	}

	// Parse the tokens with merged constants
	stmt, err := RawParse(tokens, functionDef, mergedConstants, opts)

	return stmt, functionDef, err
}
