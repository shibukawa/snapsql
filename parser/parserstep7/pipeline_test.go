package parserstep7

import (
	"testing"

	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
)

// TestPipeline_SimpleCTE tests basic CTE functionality
func TestPipeline_SimpleCTE(t *testing.T) {
	sql := `
WITH cte AS (
    SELECT id, name FROM users
)
SELECT c.id, name FROM cte c
`

	stmt := parseFullPipeline(t, sql)

	// Debug: Print all table references
	tableRefs := stmt.GetTableReferences()
	t.Logf("Total table references found: %d", len(tableRefs))

	for key, ref := range tableRefs {
		t.Logf("  Key: %s, Name: %s, RealName: %s, QueryName: %s, Context: %s", key, ref.Name, ref.RealName, ref.QueryName, ref.Context.String())
	}

	// Verify table references
	// 1. CTE "cte" uses physical table "users"
	assertHasTableReference(t, stmt, func(ref *cmn.SQTableReference) bool {
		return ref.Name == "users" && ref.RealName == "users" && ref.QueryName == "cte" && ref.Context == cmn.SQTableContextCTE
	})
	// 2. Main query uses CTE "cte" (no physical table, so RealName is empty, QueryName is "cte")
	assertHasTableReference(t, stmt, func(ref *cmn.SQTableReference) bool {
		return ref.Name == "c" && ref.RealName == "cte" && ref.QueryName == "" && ref.Context == cmn.SQTableContextMain
	})

	// Verify no dependency errors
	assertNoDependencyErrors(t, stmt)

	// Verify dependency graph nodes
	assertDependencyGraphNode(t, stmt, "cte", cmn.SQDependencyCTE)
	assertDependencyGraphNode(t, stmt, "main", cmn.SQDependencyMain)
}

// TestPipeline_SimpleFromSubquery tests basic FROM clause subquery
func TestPipeline_SimpleFromSubquery(t *testing.T) {
	sql := `
SELECT sq.id, sq.name
FROM (SELECT id, name FROM users) AS sq
`

	stmt := parseFullPipeline(t, sql)

	// Debug: Print all table references with QueryName
	tableRefs := stmt.GetTableReferences()
	t.Logf("Total table references found: %d", len(tableRefs))

	for key, ref := range tableRefs {
		t.Logf("  Key: %s, Name: %s, RealName: %s, QueryName: %s, Context: %d",
			key, ref.Name, ref.RealName, ref.QueryName, ref.Context)
	}

	// Verify subquery alias itself (in main context)
	assertHasTableReference(t, stmt, func(ref *cmn.SQTableReference) bool {
		return ref.Name == "sq" && ref.RealName == "sq" && ref.QueryName == "" && ref.Context == cmn.SQTableContextMain
	})

	// Verify the underlying users table is tracked with QueryName (in subquery context)
	assertHasTableReference(t, stmt, func(ref *cmn.SQTableReference) bool {
		return ref.Name == "users" && ref.RealName == "users" && ref.QueryName == "sq" && ref.Context == cmn.SQTableContextSubquery
	})

	// Verify no dependency errors
	assertNoDependencyErrors(t, stmt)
}

// TestPipeline_SubqueryWithMultipleTables tests subquery with multiple table references
func TestPipeline_SubqueryWithMultipleTables(t *testing.T) {
	sql := `
SELECT sq.user_id, sq.order_count
FROM (
    SELECT u.id AS user_id, COUNT(o.id) AS order_count
    FROM users u
    JOIN orders o ON u.id = o.user_id
    GROUP BY u.id
) AS sq
`

	stmt := parseFullPipeline(t, sql)

	// Verify all table references are captured
	assertHasTableReference(t, stmt, func(ref *cmn.SQTableReference) bool {
		return ref.Name == "sq" && ref.RealName == "sq" && ref.QueryName == "" && ref.Context == cmn.SQTableContextMain
	})
	assertHasTableReference(t, stmt, func(ref *cmn.SQTableReference) bool {
		return ref.Name == "u" && ref.RealName == "users" && ref.QueryName == "sq" && ref.Context == cmn.SQTableContextSubquery
	})

	// Verify no dependency errors
	assertNoDependencyErrors(t, stmt)
}

// TestPipeline_NestedSubquery tests nested subqueries
func TestPipeline_NestedSubquery(t *testing.T) {
	sql := `
SELECT outer_sq.id
FROM (
    SELECT inner_sq.id
    FROM (
        SELECT id FROM users u WHERE u.active = true
    ) AS inner_sq
) AS outer_sq
`

	stmt := parseFullPipeline(t, sql)

	// Verify all levels of table references
	assertHasTableReference(t, stmt, func(ref *cmn.SQTableReference) bool {
		return ref.Name == "outer_sq" && ref.RealName == "outer_sq" && ref.QueryName == "" && ref.Context == cmn.SQTableContextMain
	})
	assertHasTableReference(t, stmt, func(ref *cmn.SQTableReference) bool {
		return ref.Name == "inner_sq" && ref.RealName == "inner_sq" && ref.QueryName == "outer_sq" && ref.Context == cmn.SQTableContextSubquery
	})
	assertHasTableReference(t, stmt, func(ref *cmn.SQTableReference) bool {
		return ref.Name == "u" && ref.RealName == "users" && ref.QueryName == "inner_sq" && ref.Context == cmn.SQTableContextSubquery
	})

	// Verify no dependency errors
	assertNoDependencyErrors(t, stmt)
}

// TestPipeline_MultipleCTEs tests multiple CTE definitions
func TestPipeline_MultipleCTEs(t *testing.T) {
	sql := `
WITH 
    cte1 AS (SELECT u.id, name FROM users u),
    cte2 AS (SELECT o.user_id, o.product_id FROM orders o)
SELECT c1.name, c2.product_id
FROM cte1 c1
JOIN cte2 c2 ON c1.id = c2.user_id
`

	stmt := parseFullPipeline(t, sql)

	// Debug: Print all table references
	tableRefs := stmt.GetTableReferences()
	t.Logf("Total table references found: %d", len(tableRefs))

	for key, ref := range tableRefs {
		t.Logf("  Key: %s, Name: %s, RealName: %s, Context: %d", key, ref.Name, ref.RealName, ref.Context)
	}

	// Verify physical table references
	assertHasTableReference(t, stmt, func(ref *cmn.SQTableReference) bool {
		return ref.Name == "u" && ref.RealName == "users" && ref.QueryName == "cte1" && ref.Context == cmn.SQTableContextCTE
	})
	assertHasTableReference(t, stmt, func(ref *cmn.SQTableReference) bool {
		return ref.Name == "o" && ref.RealName == "orders" && ref.QueryName == "cte2" && ref.Context == cmn.SQTableContextCTE
	})

	// Verify CTE references (c1 is alias for cte1, c2 is alias for cte2)
	assertHasTableReference(t, stmt, func(ref *cmn.SQTableReference) bool {
		return ref.Name == "c1" && ref.RealName == "cte1" && ref.QueryName == "" && ref.Context == cmn.SQTableContextMain
	})
	assertHasTableReference(t, stmt, func(ref *cmn.SQTableReference) bool {
		return ref.Name == "c2" && ref.RealName == "cte2" && ref.QueryName == "" && ref.Context == cmn.SQTableContextMain
	})

	// Verify no dependency errors
	assertNoDependencyErrors(t, stmt)

	// Verify both CTEs are in the dependency graph
	assertDependencyGraphNode(t, stmt, "cte1", cmn.SQDependencyCTE)
	assertDependencyGraphNode(t, stmt, "cte2", cmn.SQDependencyCTE)
}

// TestPipeline_CTEWithDependency tests CTE that references another CTE
func TestPipeline_CTEWithDependency(t *testing.T) {
	sql := `
WITH 
    cte1 AS (SELECT id, name FROM users),
    cte2 AS (SELECT id, name FROM cte1 WHERE name LIKE 'A%')
SELECT id, name FROM cte2
`

	stmt := parseFullPipeline(t, sql)

	// Debug: Print all table references
	tableRefs := stmt.GetTableReferences()
	t.Logf("Total table references found: %d", len(tableRefs))

	for key, ref := range tableRefs {
		t.Logf("  Key: %s, Name: %s, RealName: %s, QueryName: %s, Context: %s",
			key, ref.Name, ref.RealName, ref.QueryName, ref.Context.String())
	}

	// Verify physical table reference in cte1
	assertHasTableReference(t, stmt, func(ref *cmn.SQTableReference) bool {
		return ref.Name == "users" && ref.RealName == "users" && ref.QueryName == "cte1" && ref.Context == cmn.SQTableContextCTE
	})

	// Verify CTE references usage
	assertHasTableReference(t, stmt, func(ref *cmn.SQTableReference) bool {
		return ref.Name == "cte1" && ref.RealName == "cte1" && ref.QueryName == "cte2" && ref.Context == cmn.SQTableContextCTE
	})
	assertHasTableReference(t, stmt, func(ref *cmn.SQTableReference) bool {
		return ref.Name == "cte2" && ref.RealName == "cte2" && ref.QueryName == "" && ref.Context == cmn.SQTableContextMain
	})

	// Verify no dependency errors
	assertNoDependencyErrors(t, stmt)

	// Verify processing order: users should be processed before cte1, cte1 before cte2
	// The exact order depends on implementation, but we can verify no errors
	analysis := stmt.GetSubqueryAnalysis()
	if analysis != nil && len(analysis.ProcessingOrder) > 0 {
		t.Logf("Processing order: %v", analysis.ProcessingOrder)

		// Verify cte1 comes before cte2 in processing order
		cte1Idx := -1
		cte2Idx := -1

		for i, id := range analysis.ProcessingOrder {
			if id == "cte1" {
				cte1Idx = i
			}

			if id == "cte2" {
				cte2Idx = i
			}
		}

		if cte1Idx != -1 && cte2Idx != -1 {
			if cte1Idx >= cte2Idx {
				t.Errorf("cte1 should be processed before cte2, but got order: %v", analysis.ProcessingOrder)
			}
		}
	}
}

// TestPipeline_CTEAndSubqueryCombination tests combination of CTE and subquery
func TestPipeline_CTEAndSubqueryCombination(t *testing.T) {
	sql := `
WITH active_users AS (
    SELECT id, name FROM users WHERE active = true
)
SELECT sq.id, sq.name
FROM (SELECT id, name FROM active_users) AS sq
`

	stmt := parseFullPipeline(t, sql)

	// Debug: Print all table references
	tableRefs := stmt.GetTableReferences()
	t.Logf("Total table references found: %d", len(tableRefs))

	for key, ref := range tableRefs {
		t.Logf("  Key: %s, Name: %s, RealName: %s, QueryName: %s, Context: %s",
			key, ref.Name, ref.RealName, ref.QueryName, ref.Context.String())
	}

	// Verify physical table reference
	assertHasTableReference(t, stmt, func(ref *cmn.SQTableReference) bool {
		return ref.Name == "users" && ref.RealName == "users" && ref.QueryName == "active_users" && ref.Context == cmn.SQTableContextCTE
	})

	// Verify derived table references
	assertHasTableReference(t, stmt, func(ref *cmn.SQTableReference) bool {
		return ref.Name == "active_users" && ref.RealName == "active_users" && ref.QueryName == "sq" && ref.Context == cmn.SQTableContextSubquery
	})
	assertHasTableReference(t, stmt, func(ref *cmn.SQTableReference) bool {
		return ref.Name == "sq" && ref.RealName == "sq" && ref.QueryName == "" && ref.Context == cmn.SQTableContextMain
	})

	// Verify no dependency errors
	assertNoDependencyErrors(t, stmt)
}

// TestPipeline_NoSubquery tests simple query without subqueries or CTEs
func TestPipeline_NoSubquery(t *testing.T) {
	sql := `SELECT u.id, u.name FROM users u WHERE active = true`

	stmt := parseFullPipeline(t, sql)

	// Verify single table reference
	assertHasTableReference(t, stmt, func(ref *cmn.SQTableReference) bool {
		return ref.Name == "u" && ref.RealName == "users" && ref.QueryName == "" && ref.Context == cmn.SQTableContextMain
	})

	// Verify no dependency errors
	assertNoDependencyErrors(t, stmt)

	// Analysis might not indicate subqueries, but should not have errors
	analysis := stmt.GetSubqueryAnalysis()
	if analysis != nil {
		if analysis.HasErrors {
			t.Error("simple query should not have analysis errors")
		}
	}
}
