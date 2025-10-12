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
	parser          *SubqueryParser
	errorHandler    *ErrorReporter
	subqueryCounter int // Counter for generating unique subquery IDs
}

// NewASTIntegrator creates a new AST integrator
func NewASTIntegrator(parser *SubqueryParser) *ASTIntegrator {
	return &ASTIntegrator{
		parser:          parser,
		errorHandler:    NewErrorReporter(),
		subqueryCounter: 0,
	}
}

// ExtractMainTableReferences extracts table references from the main query
// This is called before subquery extraction to ensure main query tables are recorded
func (ai *ASTIntegrator) ExtractMainTableReferences(stmt cmn.StatementNode) []*cmn.SQTableReference {
	return extractTableRefsFromStatement(stmt)
}

// ExtractSubqueries extracts subqueries from the given statement and builds dependency graph
func (ai *ASTIntegrator) ExtractSubqueries(stmt cmn.StatementNode) error {
	ai.errorHandler.Clear()

	// Create main node first
	mainNode := &cmn.SQDependencyNode{
		ID:        "main",
		Statement: stmt,
		NodeType:  cmn.SQDependencyMain,
	}
	ai.parser.dependencies.AddNode(mainNode)

	// Extract CTE from statement
	var cte *cmn.WithClause
	if selectStmt, ok := stmt.(*cmn.SelectStatement); ok {
		cte = selectStmt.CTE()
	}

	// Process different types of subqueries
	err := ai.extractCTEDependencies(cte, stmt)
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
func (ai *ASTIntegrator) extractCTEDependencies(cte *cmn.WithClause, stmt cmn.StatementNode) error {
	if cte == nil || len(cte.CTEs) == 0 {
		return nil
	}

	mainID := "main"

	// Build a map of already-processed CTEs for dependency checking
	processedCTEs := make(map[string]struct{})

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
				// Extract internal table references (with CTE context)
				tableRefs := extractTableRefsFromStatementWithCTEs(selectStmt, processedCTEs)
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
					// Extract internal table references (with CTE context)
					tableRefs := extractTableRefsFromStatementWithCTEs(selectStmt, processedCTEs)
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

		// Add internal table references from CTE's SELECT statement
		// These represent the tables used inside the CTE definition
		if cteStmt != nil {
			internalTableRefs := extractTableRefsFromStatementWithCTEs(cteStmt, processedCTEs)
			for _, internalRef := range internalTableRefs {
				// Mark these as belonging to this CTE
				internalRef.QueryName = cteDef.Name
				internalRef.Context = cmn.SQTableContextCTE
				cteNode.TableRefs = append(cteNode.TableRefs, internalRef)
			}
		}

		// Add node first before adding dependencies
		ai.parser.dependencies.AddNode(cteNode)

		// Add dependencies after node is added
		if cteStmt != nil {
			internalTableRefs := extractTableRefsFromStatementWithCTEs(cteStmt, processedCTEs)
			for _, internalRef := range internalTableRefs {
				// Check if this reference is to another CTE (RealName is empty for CTE references)
				if internalRef.RealName == "" && internalRef.Name != "" {
					// This CTE depends on another CTE
					if _, isProcessedCTE := processedCTEs[internalRef.Name]; isProcessedCTE {
						// Add dependency: referencedCTE -> currentCTE
						// (referencedCTE must be processed before currentCTE)
						ai.parser.dependencies.AddDependency(internalRef.Name, cteID)
					}
				}
			}
		}

		// Add dependency from CTE to main (CTE must be processed before main)
		ai.parser.dependencies.AddDependency(cteID, mainID)

		// Mark this CTE as processed for subsequent CTEs
		processedCTEs[cteDef.Name] = struct{}{}
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

	// Build a map of CTEs defined in this statement
	cteMap := make(map[string]struct{})

	if cte := selectStmt.CTE(); cte != nil {
		for _, cteDef := range cte.CTEs {
			cteMap[cteDef.Name] = struct{}{}
		}
	}

	// Process each table in FROM clause
	for _, table := range selectStmt.From.Tables {
		// Check if this is a subquery
		if !looksLikeSubquery(table) {
			continue
		}

		// Generate unique subquery ID
		ai.subqueryCounter++
		subqueryID := fmt.Sprintf("subquery_%d", ai.subqueryCounter)

		var (
			selectFields     []cmn.SelectField
			referencedTables []string
			subqueryStmt     cmn.StatementNode
		)

		// Parse subquery from RawTokens

		if len(table.RawTokens) > 0 {
			parsedStmt, err := parseRawTokensToSelectStatement(table.RawTokens)
			if err == nil && parsedStmt != nil {
				subqueryStmt = parsedStmt

				// Extract SelectFields from parsed SelectStatement
				if parsedSelectStmt, ok := parsedStmt.(*cmn.SelectStatement); ok {
					if parsedSelectStmt.Select != nil {
						selectFields = parsedSelectStmt.Select.Fields
					}

					// Extract internal table references
					tableRefs := extractTableRefsFromStatement(parsedStmt)
					for _, tr := range tableRefs {
						referencedTables = append(referencedTables, tr.Name)
					}

					// Recursively extract nested subqueries from this subquery
					if err := ai.extractFromClauseSubqueries(parsedStmt); err != nil {
						// Log error but continue processing
						ai.errorHandler.AddError(ErrorTypeInvalidSubquery, err.Error(), Position{})
					}
				}
			}
		}

		// Extract alias name (subquery must have an alias)
		if table.Name != "" {
			derivedTable := cmn.DerivedTableInfo{
				Name:             table.Name,
				SourceType:       "subquery",
				SelectFields:     selectFields,
				ReferencedTables: referencedTables,
			}
			ai.parser.derivedTables = append(ai.parser.derivedTables, derivedTable)

			// Create subquery dependency node with table references
			if subqueryStmt != nil {
				subqueryNode := &cmn.SQDependencyNode{
					ID:        subqueryID,
					Statement: subqueryStmt,
					NodeType:  cmn.SQDependencyFromSubquery,
				}

				// Add internal table references from subquery's SELECT statement
				// These represent the tables used inside the subquery
				internalTableRefs := extractTableRefsFromStatementWithCTEs(subqueryStmt, cteMap)
				for _, internalRef := range internalTableRefs {
					// Mark these as belonging to this subquery
					internalRef.QueryName = table.Name
					internalRef.Context = cmn.SQTableContextSubquery
					subqueryNode.TableRefs = append(subqueryNode.TableRefs, internalRef)
				}

				ai.parser.dependencies.AddNode(subqueryNode)

				// Add dependency from main to subquery
				mainID := "main"
				ai.parser.dependencies.AddDependency(mainID, subqueryID)
			}
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
	// Only populate if TableRefs is empty (to avoid overwriting already set TableRefs from ExtractSubqueries)
	if len(node.TableRefs) == 0 {
		node.TableRefs = extractTableRefsFromStatement(node.Statement)
	}

	// Field source分析は将来対応(現状は未実装)
	return nil
}

// extractTableRefsFromStatementWithCTEs extracts table references with context of already-defined CTEs
func extractTableRefsFromStatementWithCTEs(stmt cmn.StatementNode, processedCTEs map[string]struct{}) []*cmn.SQTableReference {
	var refs []*cmn.SQTableReference

	switch s := stmt.(type) {
	case *cmn.SelectStatement:
		// Create a minimal WithClause with processed CTEs for checking
		var withClause *cmn.WithClause
		if len(processedCTEs) > 0 {
			withClause = &cmn.WithClause{
				CTEs: make([]cmn.CTEDefinition, 0, len(processedCTEs)),
			}
			for name := range processedCTEs {
				withClause.CTEs = append(withClause.CTEs, cmn.CTEDefinition{Name: name})
			}
		}

		refs = append(refs, extractFromClauseTablesWithCTE(withClause, s.From)...)
	}

	return refs
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
				Context:  cmn.SQTableContextMain,
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
				Context:  cmn.SQTableContextMain,
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
				Context:  cmn.SQTableContextMain,
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
		// Heuristic: detect subquery via RawTokens or Expression tokens
		isSub := len(t.RawTokens) > 0 || looksLikeSubquery(t)

		tr := &cmn.SQTableReference{
			Name:       t.Name,
			RealName:   realName(t.TableReference),
			Schema:     t.SchemaName,
			IsSubquery: isSub,
			Join:       t.JoinType,
			Context:    cmn.SQTableContextJoin,
		}
		if i == 0 {
			tr.Join = cmn.JoinNone
			tr.Context = cmn.SQTableContextMain
		}

		if _, ok := ctes[tr.RealName]; ok {
			// CTE reference: RealName should be empty (CTE is not a physical table)
			tr.RealName = ""
		} else if tr.IsSubquery {
			// Subquery reference: RealName should be empty (subquery is not a physical table)
			// Note: The subquery alias itself remains in its current context (Main or Join)
			// Only the tables INSIDE the subquery get Context=Subquery (handled separately)
			tr.RealName = ""
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
