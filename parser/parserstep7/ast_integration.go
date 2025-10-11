package parserstep7

import (
	"errors"
	"fmt"

	snapsql "github.com/shibukawa/snapsql"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/parser/parserstep2"
	"github.com/shibukawa/snapsql/parser/parserstep3"
	"github.com/shibukawa/snapsql/parser/parserstep4"
	"github.com/shibukawa/snapsql/tokenizer"
)

// Sentinel errors
var (
	ErrSubqueryExtraction = snapsql.ErrSubqueryExtraction
	ErrNoTokensToParse    = errors.New("no tokens to parse")
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

	// Extract subqueries from FROM clause
	err = ai.extractFromClauseSubqueries(stmt)
	if err != nil {
		ai.errorHandler.AddError(ErrorTypeInvalidSubquery, err.Error(), Position{})
	}

	// Note: SELECT clause subquery extraction to be implemented in future versions

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

	// Process each CTE - extract SelectFields and ReferencedTables
	for _, cteDef := range cte.CTEs {
		cteID := cteDef.Name // Use CTE name directly without prefix/suffix

		// Parse CTE's raw tokens to get SelectStatement
		var cteStmt cmn.StatementNode

		var selectFields []cmn.SelectField

		var referencedTables []string

		// First, check if Select is already a SelectStatement (e.g., in tests)
		if selectStmt, ok := cteDef.Select.(*cmn.SelectStatement); ok && selectStmt != nil {
			cteStmt = selectStmt

			// Extract SelectFields from existing SelectStatement
			if selectStmt.Select != nil {
				selectFields = selectStmt.Select.Fields
				// Extract internal table references
				tableRefs := extractTableRefsFromStatement(selectStmt)
				for _, tr := range tableRefs {
					referencedTables = append(referencedTables, tr.Name)
				}
			}
		} else if len(cteDef.RawTokens) > 0 {
			// Use RawTokens to re-parse the CTE SELECT statement
			stmt, err := parseRawTokensToSelectStatement(cteDef.RawTokens)
			if err == nil && stmt != nil {
				cteStmt = stmt

				// Extract SelectFields from parsed SelectStatement
				if selectStmt, ok := stmt.(*cmn.SelectStatement); ok && selectStmt.Select != nil {
					selectFields = selectStmt.Select.Fields
					// Extract internal table references
					tableRefs := extractTableRefsFromStatement(selectStmt)
					for _, tr := range tableRefs {
						referencedTables = append(referencedTables, tr.Name)
					}
				}
			}
		}

		// Store extracted CTE information in parser context
		// This will be used by intermediate layer
		derivedTable := cmn.DerivedTableInfo{
			Name:             cteDef.Name,
			SourceType:       "cte",
			SelectFields:     selectFields,
			ReferencedTables: referencedTables,
		}
		ai.parser.derivedTables = append(ai.parser.derivedTables, derivedTable)

		// Create CTE dependency node
		cteNode := &cmn.SQDependencyNode{
			ID:        cteID,
			Statement: cteStmt,
			NodeType:  cmn.SQDependencyCTE,
		}
		ai.parser.dependencies.AddNode(cteNode)

		// Add dependency from main to CTE
		ai.parser.dependencies.AddDependency(mainID, cteID)
	}

	return nil
}

// extractFromClauseSubqueries extracts subqueries from FROM clause
func (ai *ASTIntegrator) extractFromClauseSubqueries(stmt cmn.StatementNode) error {
	// Only process SELECT statements with FROM clause
	selectStmt, ok := stmt.(*cmn.SelectStatement)
	if !ok || selectStmt.From == nil {
		return nil
	}

	// Process each table in FROM clause
	for _, table := range selectStmt.From.Tables {
		// Check if this is a subquery
		if !table.IsSubquery && !looksLikeSubquery(table) {
			continue
		}

		// Try to parse the subquery expression
		// For now, we'll create a placeholder entry
		// Full implementation would parse the subquery tokens into a StatementNode
		if table.Name != "" {
			// Extract alias name (subquery must have an alias)
			derivedTable := cmn.DerivedTableInfo{
				Name:             table.Name,
				SourceType:       "subquery",
				SelectFields:     []cmn.SelectField{}, // TODO: Parse subquery to extract fields
				ReferencedTables: []string{},          // TODO: Parse subquery to extract tables
			}
			ai.parser.derivedTables = append(ai.parser.derivedTables, derivedTable)
		}
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

// parseRawTokensToSelectStatement attempts to re-parse raw tokens into a SelectStatement
// This is used to parse CTE SELECT statements from their raw token representation
func parseRawTokensToSelectStatement(rawTokens []tokenizer.Token) (cmn.StatementNode, error) {
	if len(rawTokens) == 0 {
		return nil, ErrNoTokensToParse
	}

	// CTE tokens include surrounding parentheses: ( SELECT ... )
	// We need to remove them before parsing
	tokens := rawTokens
	if len(tokens) > 0 && tokens[0].Type == tokenizer.OPENED_PARENS {
		// Remove first and last tokens (parentheses)
		if len(tokens) >= 2 && tokens[len(tokens)-1].Type == tokenizer.CLOSED_PARENS {
			tokens = tokens[1 : len(tokens)-1]
		}
	}

	// Step 1: Use parserstep2.Execute to parse the tokens into StatementNode with clauses
	stmt, err := parserstep2.Execute(tokens)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CTE tokens (step2): %w", err)
	}

	// Step 2: Use parserstep3.Execute to assign clauses to statement fields
	err = parserstep3.Execute(stmt)
	if err != nil {
		return nil, fmt.Errorf("failed to assign clauses (step3): %w", err)
	}

	// Step 3: Use parserstep4.Execute to finalize and validate clauses
	err = parserstep4.Execute(stmt)
	if err != nil {
		return nil, fmt.Errorf("failed to finalize clauses (step4): %w", err)
	}

	return stmt, nil
}
