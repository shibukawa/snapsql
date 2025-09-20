package parserstep7

import (
	"fmt"

	snapsql "github.com/shibukawa/snapsql"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
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

	// Populate table references for this node from the AST
	node.TableRefs = extractTableRefsFromStatement(node.Statement)

	// Field source分析は将来対応（現状は未実装）
	return nil
}

// extractTableRefsFromStatement collects tables referenced by the statement.
// It covers SELECT (FROM), INSERT (INTO + FROM), UPDATE (target), DELETE (target).
func extractTableRefsFromStatement(stmt cmn.StatementNode) []*cmn.SQTableReference {
	var refs []*cmn.SQTableReference

	switch s := stmt.(type) {
	case *cmn.SelectStatement:
		refs = append(refs, extractFromClauseTablesWithCTE(s.CTE(), s.From)...)
	case *cmn.InsertIntoStatement:
		if s.Into != nil {
			tr := &cmn.SQTableReference{
				Name:     nameOrAlias(s.Into.Table),
				RealName: realName(s.Into.Table),
				Schema:   s.Into.Table.SchemaName,
				Join:     cmn.JoinNone,
				Source:   cmn.SQTableSourceMain,
			}
			refs = append(refs, tr)
		}

		refs = append(refs, extractFromClauseTablesWithCTE(s.CTE(), s.From)...)
	case *cmn.UpdateStatement:
		if s.Update != nil {
			tr := &cmn.SQTableReference{
				Name:     nameOrAlias(s.Update.Table),
				RealName: realName(s.Update.Table),
				Schema:   s.Update.Table.SchemaName,
				Join:     cmn.JoinNone,
				Source:   cmn.SQTableSourceMain,
			}
			refs = append(refs, tr)
		}
	case *cmn.DeleteFromStatement:
		if s.From != nil {
			tr := &cmn.SQTableReference{
				Name:     nameOrAlias(s.From.Table),
				RealName: realName(s.From.Table),
				Schema:   s.From.Table.SchemaName,
				Join:     cmn.JoinNone,
				Source:   cmn.SQTableSourceMain,
			}
			refs = append(refs, tr)
		}
	}

	return refs
}

func extractFromClauseTablesWithCTE(with *cmn.WithClause, from *cmn.FromClause) []*cmn.SQTableReference {
	if from == nil {
		return nil
	}
	// Build CTE set
	ctes := map[string]struct{}{}

	if with != nil {
		for _, d := range with.CTEs {
			if d.Name != "" {
				ctes[d.Name] = struct{}{}
			}
		}
	}

	out := make([]*cmn.SQTableReference, 0, len(from.Tables))

	for i, t := range from.Tables {
		// Heuristic: some earlier steps don't flag IsSubquery; detect via Expression tokens
		isSub := t.IsSubquery || looksLikeSubquery(t)

		tr := &cmn.SQTableReference{
			Name:       t.Name,
			RealName:   realName(t.TableReference),
			Schema:     t.SchemaName,
			IsSubquery: isSub,
			Join:       t.JoinType,
			Source:     cmn.SQTableSourceJoin,
		}
		if i == 0 {
			tr.Join = cmn.JoinNone
			tr.Source = cmn.SQTableSourceMain
		}

		if _, ok := ctes[tr.RealName]; ok {
			tr.Source = cmn.SQTableSourceCTE
		} else if tr.IsSubquery {
			tr.Source = cmn.SQTableSourceSubquery
		}

		out = append(out, tr)
	}

	return out
}

func looksLikeSubquery(t cmn.TableReferenceForFrom) bool {
	// Quick heuristic: if no base table name, likely a derived table
	if t.TableName == "" {
		return true
	}
	// If we have expression tokens starting with '(' and containing SELECT, treat as subquery
	if len(t.Expression) == 0 {
		return false
	}

	hasParen := false
	hasSelect := false

	for _, tok := range t.Expression {
		switch tok.Type {
		case tokenizer.OPENED_PARENS:
			hasParen = true
		case tokenizer.SELECT:
			hasSelect = true
		}

		if hasParen && hasSelect {
			return true
		}
	}

	return false
}

func nameOrAlias(t cmn.TableReference) string { // alias if exists else name
	if t.TableName != "" && t.Name != "" && t.TableName != t.Name {
		return t.Name
	}

	if t.Name != "" {
		return t.Name
	}

	return t.TableName
}

func realName(t cmn.TableReference) string {
	if t.TableName != "" {
		return t.TableName
	}

	return t.Name
}

// GetDependencyGraph returns the built dependency graph
func (ai *ASTIntegrator) GetDependencyGraph() *cmn.SQDependencyGraph {
	return ai.parser.dependencies
}

// GetErrors returns any errors encountered during processing
func (ai *ASTIntegrator) GetErrors() []*cmn.SQParseError {
	return ai.errorHandler.GetErrors()
}
