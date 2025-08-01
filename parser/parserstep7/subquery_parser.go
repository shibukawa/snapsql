package parserstep7

import (
	"errors"

	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
)

// SubqueryParser handles subquery parsing and dependency resolution
type SubqueryParser struct {
	idGenerator  *IDGenerator
	dependencies *cmn.SQDependencyGraph
}

// NewSubqueryParser creates a new subquery parser
func NewSubqueryParser() *SubqueryParser {
	return &SubqueryParser{
		idGenerator:  NewIDGenerator(),
		dependencies: NewDependencyGraph(),
	}
}

// ParseSubqueries parses subqueries and builds field source information
func (sp *SubqueryParser) ParseSubqueries(stmt cmn.StatementNode) error {
	// 1. Initialize dependency graph
	sp.dependencies = NewDependencyGraph()
	sp.idGenerator.ResetAll()

	// 2. Build dependency graph by detecting subqueries
	if err := sp.buildDependencyGraph(stmt); err != nil {
		return err
	}

	// 3. Check for circular dependencies
	if cmn.HasCircularDependencyInGraph(sp.dependencies) {
		return ErrCircularDependency
	}

	// 4. Determine processing order (topological sort)
	processingOrder, err := cmn.GetProcessingOrderFromGraph(sp.dependencies)
	if err != nil {
		return err
	}

	// 5. Parse subqueries in dependency order
	for _, nodeID := range processingOrder {
		if err := sp.parseSubqueryNode(nodeID); err != nil {
			return err
		}
	}

	// 6. Build field source information
	if err := sp.buildFieldSources(stmt); err != nil {
		return err
	}

	return nil
}

// buildDependencyGraph builds the dependency graph for subqueries
func (sp *SubqueryParser) buildDependencyGraph(stmt cmn.StatementNode) error {
	mainID := sp.idGenerator.Generate("main")
	mainNode := &cmn.SQDependencyNode{
		ID:        mainID,
		Statement: stmt,
		NodeType:  cmn.SQDependencyMain,
	}
	sp.dependencies.AddNode(mainNode)

	// Build dependencies for WITH clauses
	if cte := stmt.CTE(); cte != nil {
		for _, cteDef := range cte.CTEs {
			cteID := sp.idGenerator.Generate("cte_" + cteDef.Name)

			// CTEDefinition.Select is AstNode, need to check if it's StatementNode
			if cteStmt, ok := cteDef.Select.(cmn.StatementNode); ok {
				cteNode := &cmn.SQDependencyNode{
					ID:        cteID,
					Statement: cteStmt,
					NodeType:  cmn.SQDependencyCTE,
				}
				sp.dependencies.AddNode(cteNode)

				// Recursively analyze dependencies in CTE subquery
				if err := sp.analyzeDependenciesInStatement(cteStmt, cteID); err != nil {
					return err
				}
			}
		}
	}

	// Build dependencies for each clause
	for _, clause := range stmt.Clauses() {
		if err := sp.buildClauseDependencies(clause, mainID); err != nil {
			return err
		}
	}

	return nil
}

// buildClauseDependencies builds dependencies for a specific clause
func (sp *SubqueryParser) buildClauseDependencies(clause cmn.ClauseNode, parentID string) error {
	switch c := clause.(type) {
	case *cmn.SelectClause:
		return sp.buildSelectFieldDependencies(c.Fields, parentID)
	case *cmn.FromClause:
		return sp.buildFromDependencies(c.Tables, parentID)
		// Note: WHERE and HAVING clauses are excluded (no impact on type inference)
	}
	return nil
}

// buildSelectFieldDependencies builds dependencies for SELECT fields
func (sp *SubqueryParser) buildSelectFieldDependencies(fields []cmn.SelectField, parentID string) error {
	// Current SelectField structure doesn't have Subquery field
	// This would need to be implemented when subquery support is added to SelectField
	// For now, return nil
	return nil
}

// buildFromDependencies builds dependencies for FROM clause tables
func (sp *SubqueryParser) buildFromDependencies(tables []cmn.TableReferenceForFrom, parentID string) error {
	for _, table := range tables {
		if table.IsSubquery {
			subqueryID := sp.idGenerator.Generate("from_subquery")
			// TableReferenceForFrom doesn't have Subquery field yet
			// This would need to be implemented when subquery support is added
			_ = subqueryID
		}
	}
	return nil
}

// analyzeDependenciesInStatement recursively analyzes dependencies in a statement
func (sp *SubqueryParser) analyzeDependenciesInStatement(stmt cmn.StatementNode, parentID string) error {
	// This would recursively call buildDependencyGraph for nested subqueries
	// For now, we'll implement a simplified version
	return nil
}

// parseSubqueryNode parses a specific subquery node
func (sp *SubqueryParser) parseSubqueryNode(nodeID string) error {
	node := sp.dependencies.GetNode(nodeID)
	if node == nil {
		return errors.New("node not found: " + nodeID)
	}

	// Here we would perform detailed parsing of the subquery
	// For now, we'll implement a placeholder
	return nil
}

// buildFieldSources builds field source information for the statement
func (sp *SubqueryParser) buildFieldSources(stmt cmn.StatementNode) error {
	fieldSources := make(map[string]*FieldSource)
	tableReferences := make(map[string]*TableReference)

	// 1. Build table references
	if err := sp.buildTableReferences(stmt, tableReferences); err != nil {
		return err
	}

	// 2. Build field sources
	if err := sp.buildSelectFieldSources(stmt, fieldSources, tableReferences); err != nil {
		return err
	}

	// 3. Convert to interface{} types and set results in StatementNode
	interfaceFieldSources := make(map[string]*cmn.SQFieldSource)
	for k, v := range fieldSources {
		interfaceFieldSources[k] = v
	}

	interfaceTableReferences := make(map[string]*cmn.SQTableReference)
	for k, v := range tableReferences {
		interfaceTableReferences[k] = v
	}

	cmn.SetFieldSources(stmt, interfaceFieldSources)
	cmn.SetTableReferences(stmt, interfaceTableReferences)
	cmn.SetSubqueryDependencies(stmt, sp.dependencies)

	return nil
}

// buildTableReferences builds table reference information
func (sp *SubqueryParser) buildTableReferences(stmt cmn.StatementNode, tableRefs map[string]*TableReference) error {
	// Implementation placeholder
	return nil
}

// buildSelectFieldSources builds field source information from SELECT clause
func (sp *SubqueryParser) buildSelectFieldSources(stmt cmn.StatementNode, fieldSources map[string]*FieldSource, tableRefs map[string]*TableReference) error {
	for _, clause := range stmt.Clauses() {
		if selectClause, ok := clause.(*cmn.SelectClause); ok {
			for _, field := range selectClause.Fields {
				source := &FieldSource{
					Name:  field.FieldName,
					Alias: field.FieldName, // SelectField doesn't have separate alias field
				}

				// Determine source type based on available fields
				if field.FieldKind == cmn.FunctionField {
					source.SourceType = SourceTypeExpression
					source.ExprSource = field.OriginalField
				} else if field.TableName != "" {
					source.SourceType = SourceTypeTable
					source.TableSource = sp.resolveTableSource(field.TableName, field.FieldName, tableRefs)
				} else {
					source.SourceType = SourceTypeLiteral
					source.ExprSource = field.OriginalField
				}

				fieldKey := field.FieldName
				if fieldKey == "" {
					fieldKey = field.OriginalField
				}
				fieldSources[fieldKey] = source
			}
		}
	}
	return nil
}

// Helper methods
func (sp *SubqueryParser) getSubqueryID(subquery cmn.StatementNode) string {
	// Implementation placeholder
	return ""
}

func (sp *SubqueryParser) isExpression(fieldKind cmn.FieldType) bool {
	return fieldKind == cmn.FunctionField
}

func (sp *SubqueryParser) resolveTableSource(tableName, fieldName string, tableRefs map[string]*TableReference) *TableReference {
	// Try to find existing table reference or create a new one
	if ref, exists := tableRefs[tableName]; exists {
		return ref
	}

	// Create new table reference
	tableRef := &TableReference{
		Name:     tableName,
		RealName: tableName,
		Fields:   []*FieldSource{},
	}
	tableRefs[tableName] = tableRef
	return tableRef
}
