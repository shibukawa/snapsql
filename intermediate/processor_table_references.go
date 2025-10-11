package intermediate

import (
	"github.com/shibukawa/snapsql/parser"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
)

// TableReferencesProcessor extracts table reference information from the statement
type TableReferencesProcessor struct{}

func (p *TableReferencesProcessor) Name() string {
	return "TableReferencesProcessor"
}

func (p *TableReferencesProcessor) Process(ctx *ProcessingContext) error {
	// Extract table references from statement
	tableRefs := extractTableReferences(ctx.Statement)

	// Store in context
	ctx.TableReferences = tableRefs

	return nil
}

// extractTableReferences extracts all table references from a statement
func extractTableReferences(stmt parser.StatementNode) []TableReferenceInfo {
	var refs []TableReferenceInfo

	// Get subquery analysis result (contains DerivedTables)
	if stmt.HasSubqueryAnalysis() {
		analysis := stmt.GetSubqueryAnalysis()
		if analysis != nil {
			// Extract CTE and subquery references from DerivedTables
			for _, dt := range analysis.DerivedTables {
				// Add the CTE/subquery itself as a table reference
				ref := TableReferenceInfo{
					Name:      dt.Name,
					TableName: "", // CTEs and subqueries don't have original table names
					Alias:     dt.Name,
					Context:   dt.SourceType, // "cte" or "subquery"
				}
				refs = append(refs, ref)

				// Note: ReferencedTables (internal tables within CTE/subquery) are NOT
				// added as separate table references. They are implementation details
				// of the CTE/subquery and should not appear in the table_references output.
			}
		}
	}

	// Extract main query table references
	refs = append(refs, extractMainTableReferences(stmt)...)

	return refs
}

// extractMainTableReferences extracts table references from main query (not CTE internals)
func extractMainTableReferences(stmt parser.StatementNode) []TableReferenceInfo {
	var refs []TableReferenceInfo

	// For SELECT statements, extract only from the main query's FROM/JOIN clauses
	// This excludes tables within CTE definitions
	selectStmt, ok := stmt.(*cmn.SelectStatement)
	if !ok {
		return refs
	}

	// Extract from FROM clause
	if selectStmt.From != nil {
		for _, table := range selectStmt.From.Tables {
			// Skip CTE references (they're already added from DerivedTables)
			// We only want actual table references here
			if table.IsSubquery {
				continue // Subqueries handled separately
			}

			ref := TableReferenceInfo{
				Name:      table.Name,
				TableName: table.TableName,
				Context:   "main",
			}

			// Set alias if different from table name
			if table.Name != table.TableName && table.TableName != "" {
				ref.Alias = table.Name
			}

			refs = append(refs, ref)
		}
	}

	return refs
}
