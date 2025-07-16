package parserstep7

import (
	"fmt"

	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/tokenizer"
)

// demoStatementNode is a minimal implementation for demo purposes
type demoStatementNode struct {
	cte                  *cmn.WithClause
	fieldSources         map[string]cmn.FieldSourceInterface
	tableReferences      map[string]cmn.TableReferenceInterface
	subqueryDependencies cmn.DependencyGraphInterface
}

func (d *demoStatementNode) CTE() *cmn.WithClause             { return d.cte }
func (d *demoStatementNode) LeadingTokens() []tokenizer.Token { return nil }
func (d *demoStatementNode) Clauses() []cmn.ClauseNode        { return nil }
func (d *demoStatementNode) GetFieldSources() map[string]cmn.FieldSourceInterface {
	if d.fieldSources == nil {
		d.fieldSources = make(map[string]cmn.FieldSourceInterface)
	}
	return d.fieldSources
}
func (d *demoStatementNode) GetTableReferences() map[string]cmn.TableReferenceInterface {
	if d.tableReferences == nil {
		d.tableReferences = make(map[string]cmn.TableReferenceInterface)
	}
	return d.tableReferences
}
func (d *demoStatementNode) GetSubqueryDependencies() cmn.DependencyGraphInterface {
	return d.subqueryDependencies
}
func (d *demoStatementNode) SetFieldSources(fs map[string]cmn.FieldSourceInterface) {
	d.fieldSources = fs
}
func (d *demoStatementNode) SetTableReferences(tr map[string]cmn.TableReferenceInterface) {
	d.tableReferences = tr
}
func (d *demoStatementNode) SetSubqueryDependencies(dg cmn.DependencyGraphInterface) {
	d.subqueryDependencies = dg
}
func (d *demoStatementNode) FindFieldReference(tableOrAlias, fieldOrReference string) cmn.FieldSourceInterface {
	return nil
}
func (d *demoStatementNode) FindTableReference(tableOrAlias string) cmn.TableReferenceInterface {
	return nil
}
func (d *demoStatementNode) Position() tokenizer.Position { return tokenizer.Position{} }
func (d *demoStatementNode) RawTokens() []tokenizer.Token { return nil }
func (d *demoStatementNode) String() string               { return "demo_statement" }
func (d *demoStatementNode) Type() cmn.NodeType           { return cmn.SELECT_STATEMENT }

// Implement new StatementNode interface methods
func (d *demoStatementNode) GetSubqueryAnalysis() *cmn.SubqueryAnalysisInfo {
	return nil // Demo implementation returns nil
}
func (d *demoStatementNode) SetSubqueryAnalysis(info *cmn.SubqueryAnalysisInfo) {
	// Demo implementation does nothing
}
func (d *demoStatementNode) HasSubqueryAnalysis() bool {
	return false // Demo implementation always returns false
}

// DemoFieldSourceManagement demonstrates the field source management capabilities
func DemoFieldSourceManagement() {
	fmt.Println("=== parserstep7 Field Source Management Demo ===")

	// Create an integrated parser
	parser := NewSubqueryParserIntegrated()

	// Create a mock statement with CTE
	fmt.Println("\n1. Creating sample SQL structure:")
	fmt.Println("   WITH user_stats AS (SELECT id, name, COUNT(*) as total FROM users GROUP BY id, name)")
	fmt.Println("   SELECT us.id, us.total * 2 as doubled FROM user_stats us")

	cte := &cmn.CTEDefinition{
		Name: "user_stats",
		Select: &demoStatementNode{
			cte: &cmn.WithClause{
				CTEs: []cmn.CTEDefinition{},
			},
		},
	}

	mainStmt := &demoStatementNode{
		cte: &cmn.WithClause{
			CTEs: []cmn.CTEDefinition{*cte},
		},
	}

	// Parse the statement
	fmt.Println("\n2. Parsing statement and building dependency graph...")
	result, err := parser.ParseStatement(mainStmt)
	if err != nil {
		fmt.Printf("Error parsing statement: %v\n", err)
		return
	}

	fmt.Printf("   - Dependency graph created with %d nodes\n", len(result.DependencyGraph.GetAllNodes()))
	fmt.Printf("   - Processing order: %v\n", result.ProcessingOrder)

	// Get node IDs
	var cteNodeID, mainNodeID string
	nodes := result.DependencyGraph.GetAllNodes()

	for nodeID, node := range nodes {
		switch node.NodeType {
		case DependencyCTE:
			cteNodeID = nodeID
		case DependencyMain:
			mainNodeID = nodeID
		}
	}

	if cteNodeID == "" || mainNodeID == "" {
		fmt.Println("   Warning: Expected CTE and Main nodes not found")
		// Use first two nodes for demo
		count := 0
		for nodeID := range nodes {
			if count == 0 {
				cteNodeID = nodeID
			} else if count == 1 {
				mainNodeID = nodeID
			}
			count++
			if count >= 2 {
				break
			}
		}
	}

	// Add field sources to CTE node
	fmt.Println("\n3. Adding field sources to CTE node...")
	cteFields := []*FieldSource{
		{Name: "id", SourceType: SourceTypeTable, Scope: cteNodeID},
		{Name: "name", SourceType: SourceTypeTable, Scope: cteNodeID},
		{Name: "total", SourceType: SourceTypeAggregate, Scope: cteNodeID, ExprSource: "COUNT(*)"},
	}

	for _, field := range cteFields {
		err = parser.AddFieldSourceToNode(cteNodeID, field)
		if err != nil {
			fmt.Printf("   Error adding field %s: %v\n", field.Name, err)
		} else {
			fmt.Printf("   + Added field: %s (%s)\n", field.Name, field.SourceType.String())
		}
	}

	// Add table reference to CTE
	cteTableRef := &TableReference{
		Name:     "users",
		RealName: "users",
		Schema:   "public",
	}
	err = parser.AddTableReferenceToNode(cteNodeID, cteTableRef)
	if err != nil {
		fmt.Printf("   Error adding table reference: %v\n", err)
	} else {
		fmt.Printf("   + Added table reference: users\n")
	}

	// Add field sources to main query node
	fmt.Println("\n4. Adding field sources to main query node...")
	mainFields := []*FieldSource{
		{Name: "doubled", SourceType: SourceTypeExpression, Scope: mainNodeID, ExprSource: "us.total * 2"},
	}

	for _, field := range mainFields {
		err = parser.AddFieldSourceToNode(mainNodeID, field)
		if err != nil {
			fmt.Printf("   Error adding field %s: %v\n", field.Name, err)
		} else {
			fmt.Printf("   + Added field: %s (%s)\n", field.Name, field.SourceType.String())
		}
	}

	// Add table alias for CTE
	mainTableRef := &TableReference{
		Name:       "us",
		RealName:   "user_stats",
		Schema:     "",
		IsSubquery: true,
		SubqueryID: cteNodeID,
	}
	err = parser.AddTableReferenceToNode(mainNodeID, mainTableRef)
	if err != nil {
		fmt.Printf("   Error adding table alias: %v\n", err)
	} else {
		fmt.Printf("   + Added table alias: us -> user_stats\n")
	}

	// Test field resolution
	fmt.Println("\n5. Testing field resolution...")

	// Test resolving CTE fields from main query
	fmt.Println("   Resolving fields from main query perspective:")
	testFields := []string{"id", "total", "doubled"}

	for _, fieldName := range testFields {
		resolved, err := parser.ResolveFieldReference(mainNodeID, fieldName)
		if err != nil {
			fmt.Printf("   - %s: ❌ %v\n", fieldName, err)
		} else {
			fmt.Printf("   - %s: ✅ found %d source(s)\n", fieldName, len(resolved))
			for _, source := range resolved {
				fmt.Printf("     └─ %s (%s)\n", source.Name, source.SourceType.String())
			}
		}
	}

	// Test field access validation
	fmt.Println("\n6. Testing field access validation...")
	fmt.Println("   Validating field access from main query:")

	validationTests := []string{"id", "total", "doubled", "nonexistent_field"}
	for _, fieldName := range validationTests {
		err = parser.ValidateFieldAccess(mainNodeID, fieldName)
		if err != nil {
			fmt.Printf("   - %s: ❌ %v\n", fieldName, err)
		} else {
			fmt.Printf("   - %s: ✅ accessible\n", fieldName)
		}
	}

	// Get all accessible fields
	fmt.Println("\n7. Getting all accessible fields...")
	accessibleFields, err := parser.GetAccessibleFieldsForNode(mainNodeID)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		fmt.Printf("   Fields accessible from main query (%d total):\n", len(accessibleFields))
		for _, field := range accessibleFields {
			fmt.Printf("   - %s (%s) from scope %s\n", field.Name, field.SourceType.String(), field.Scope)
		}
	}

	// Display scope hierarchy
	fmt.Println("\n8. Scope hierarchy visualization:")
	scopeHierarchy := parser.GetScopeVisualization()
	lines := splitLines(scopeHierarchy)
	for _, line := range lines {
		fmt.Printf("   %s\n", line)
	}

	// Display dependency visualization
	fmt.Println("\n9. Dependency graph visualization:")
	depVisualization := parser.GetDependencyVisualization()
	lines = splitLines(depVisualization)
	for _, line := range lines {
		fmt.Printf("   %s\n", line)
	}

	fmt.Println("\n=== Demo completed successfully ===")
}

// Helper function to split strings into lines
func splitLines(text string) []string {
	if text == "" {
		return []string{}
	}

	var lines []string
	current := ""

	for _, char := range text {
		if char == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(char)
		}
	}

	if current != "" {
		lines = append(lines, current)
	}

	return lines
}

// Alternative demo showing error cases
func DemoFieldSourceErrorCases() {
	fmt.Println("\n=== Field Source Management Error Cases Demo ===")

	parser := NewSubqueryParserIntegrated()

	fmt.Println("\n1. Testing operations before parsing any statement:")

	// Test operations without parsing
	err := parser.ValidateFieldAccess("nonexistent", "field")
	fmt.Printf("   ValidateFieldAccess: %v\n", err)

	_, err = parser.ResolveFieldReference("nonexistent", "field")
	fmt.Printf("   ResolveFieldReference: %v\n", err)

	_, err = parser.GetAccessibleFieldsForNode("nonexistent")
	fmt.Printf("   GetAccessibleFieldsForNode: %v\n", err)

	fmt.Println("\n2. Testing operations on non-existent nodes:")

	// Parse a simple statement first
	stmt := &demoStatementNode{
		cte: &cmn.WithClause{
			CTEs: []cmn.CTEDefinition{},
		},
	}

	_, err = parser.ParseStatement(stmt)
	if err != nil {
		fmt.Printf("   Parse error: %v\n", err)
		return
	}

	// Test operations on non-existent node
	fieldSource := &FieldSource{Name: "test", SourceType: SourceTypeTable}
	err = parser.AddFieldSourceToNode("invalid_node", fieldSource)
	fmt.Printf("   AddFieldSourceToNode: %v\n", err)

	tableRef := &TableReference{Name: "t", RealName: "test"}
	err = parser.AddTableReferenceToNode("invalid_node", tableRef)
	fmt.Printf("   AddTableReferenceToNode: %v\n", err)

	fmt.Println("\n=== Error cases demo completed ===")
}
