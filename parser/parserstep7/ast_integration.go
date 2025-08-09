package parserstep7

import (
	"fmt"

	snapsql "github.com/shibukawa/snapsql"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
)

// Sentinel errors
var (
	ErrSubqueryExtraction = snapsql.ErrSubqueryExtraction
)

// ASTIntegrator integrates with actual SQL AST structures to detect and parse subqueries
type ASTIntegrator struct {
	parser       *SubqueryParser
	errorHandler *ErrorReporter
}

// NewASTIntegrator creates a new AST integrator
func NewASTIntegrator(parser *SubqueryParser) *ASTIntegrator {
	return &ASTIntegrator{
		parser:       parser,
		errorHandler: NewErrorReporter(),
	}
}

// ExtractSubqueries extracts subqueries from the given statement and builds dependency graph
func (ai *ASTIntegrator) ExtractSubqueries(stmt cmn.StatementNode) error {
	ai.errorHandler.Clear()

	// Process different types of subqueries
	err := ai.extractCTEDependencies(stmt)
	if err != nil {
		ai.errorHandler.AddError(ErrorTypeInvalidSubquery, err.Error(), Position{})
	}

	// Note: FROM clause and SELECT clause subquery extraction to be implemented in future versions

	if ai.errorHandler.HasErrors() {
		return ErrSubqueryExtraction
	}

	return nil
}

// extractCTEDependencies extracts WITH clause CTEs and builds their dependencies
func (ai *ASTIntegrator) extractCTEDependencies(stmt cmn.StatementNode) error {
	cte := stmt.CTE()
	if cte == nil {
		return nil
	}

	// Create main statement node
	mainID := ai.parser.idGenerator.Generate("main")
	mainNode := &cmn.SQDependencyNode{
		ID:        mainID,
		Statement: stmt,
		NodeType:  cmn.SQDependencyMain,
	}
	ai.parser.dependencies.AddNode(mainNode)

	// Process each CTE - register them as dependency nodes
	// Note: Future implementation would recursively parse each CTE's Select statement
	for _, cteDef := range cte.CTEs {
		cteID := ai.parser.idGenerator.Generate("cte_" + cteDef.Name)

		// Create placeholder StatementNode for the CTE
		// Note: Full implementation would parse cteDef.Select as StatementNode
		cteNode := &cmn.SQDependencyNode{
			ID:        cteID,
			Statement: nil, // Placeholder for future CTE statement parsing
			NodeType:  cmn.SQDependencyCTE,
		}
		ai.parser.dependencies.AddNode(cteNode)

		// Add dependency from main to CTE
		ai.parser.dependencies.AddDependency(mainID, cteID)
	}

	return nil
}

// BuildFieldSources builds field source information for all nodes
func (ai *ASTIntegrator) BuildFieldSources() error {
	// Get processing order
	order, err := ai.parser.dependencies.GetProcessingOrder()
	if err != nil {
		return fmt.Errorf("failed to get processing order: %w", err)
	}

	// Build field sources in dependency order
	for _, nodeID := range order {
		node := ai.parser.dependencies.GetNode(nodeID)
		if node == nil {
			continue
		}

		err := ai.buildNodeFieldSources(node)
		if err != nil {
			return fmt.Errorf("failed to build field sources for node %s: %w", nodeID, err)
		}
	}

	return nil
}

// buildNodeFieldSources builds field source information for a single node
func (ai *ASTIntegrator) buildNodeFieldSources(node *cmn.SQDependencyNode) error {
	if node.Statement == nil {
		return nil // Skip nodes without statements (e.g., CTE placeholders)
	}

	selectStmt, ok := node.Statement.(*cmn.SelectStatement)
	if !ok {
		return nil // Only SELECT statements produce fields
	}

	if selectStmt.Select == nil {
		return nil
	}

	var fieldSources []*FieldSource

	// Create basic field sources without detailed analysis
	// Note: Full implementation would analyze actual field expressions
	fieldSources = append(fieldSources, &FieldSource{
		Name:       "placeholder_field",
		SourceType: SourceTypeTable,
	})

	// Store field sources in the node (requires extending DependencyNode structure)
	// Note: Future version will add FieldSources field to DependencyNode
	_ = fieldSources

	return nil
}

// GetDependencyGraph returns the built dependency graph
func (ai *ASTIntegrator) GetDependencyGraph() *cmn.SQDependencyGraph {
	return ai.parser.dependencies
}

// GetErrors returns any errors encountered during processing
func (ai *ASTIntegrator) GetErrors() []*cmn.SQParseError {
	return ai.errorHandler.GetErrors()
}
