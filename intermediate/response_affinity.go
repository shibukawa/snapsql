package intermediate

import (
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/parser/parsercommon"
)

// ResponseAffinity represents the cardinality of the query result
type ResponseAffinity string

const (
	// ResponseAffinityOne indicates the query returns a single row
	ResponseAffinityOne ResponseAffinity = "one"

	// ResponseAffinityMany indicates the query returns multiple rows
	ResponseAffinityMany ResponseAffinity = "many"

	// ResponseAffinityNone indicates the query doesn't return any rows (e.g., INSERT, UPDATE, DELETE)
	ResponseAffinityNone ResponseAffinity = "none"
)

// DetermineResponseAffinity analyzes the statement and determines the response affinity
func DetermineResponseAffinity(stmt parser.StatementNode) ResponseAffinity {
	// Default affinity is "many" for SELECT statements
	affinity := ResponseAffinityMany

	// Determine affinity based on statement type
	switch stmt.Type() {
	case parsercommon.SELECT_STATEMENT:
		// For SELECT statements, check if it's a single row query
		selectStmt, ok := stmt.(*parsercommon.SelectStatement)
		if ok {
			// Check if it has LIMIT 1
			if hasLimit1(selectStmt) {
				affinity = ResponseAffinityOne
			} else if hasUniqueKeyCondition(selectStmt) {
				// Check if it's a single row query (e.g., has a WHERE clause with a primary key)
				affinity = ResponseAffinityOne
			} else {
				affinity = ResponseAffinityMany
			}
		}

	case parsercommon.INSERT_INTO_STATEMENT:
		// For INSERT statements, check if it has a RETURNING clause
		insertStmt, ok := stmt.(*parsercommon.InsertIntoStatement)
		if ok && insertStmt.Returning != nil {
			// For INSERT with RETURNING, determine if it's a single row or multiple rows
			if isBulkInsert(insertStmt) {
				affinity = ResponseAffinityMany
			} else {
				affinity = ResponseAffinityOne
			}
		} else {
			// INSERT without RETURNING doesn't return rows
			affinity = ResponseAffinityNone
		}

	case parsercommon.UPDATE_STATEMENT:
		// For UPDATE statements, check if it has a RETURNING clause
		updateStmt, ok := stmt.(*parsercommon.UpdateStatement)
		if ok && updateStmt.Returning != nil {
			// UPDATE with RETURNING typically returns multiple rows
			affinity = ResponseAffinityMany
		} else {
			// UPDATE without RETURNING doesn't return rows
			affinity = ResponseAffinityNone
		}

	case parsercommon.DELETE_FROM_STATEMENT:
		// For DELETE statements, check if it has a RETURNING clause
		deleteStmt, ok := stmt.(*parsercommon.DeleteFromStatement)
		if ok && deleteStmt.Returning != nil {
			// DELETE with RETURNING typically returns multiple rows
			affinity = ResponseAffinityMany
		} else {
			// DELETE without RETURNING doesn't return rows
			affinity = ResponseAffinityNone
		}

	default:
		// For other statement types, default to "none"
		affinity = ResponseAffinityNone
	}

	return affinity
}

// hasUniqueKeyCondition checks if a SELECT statement has a WHERE clause with a unique key condition
func hasUniqueKeyCondition(stmt *parsercommon.SelectStatement) bool {
	// This is a placeholder implementation
	// In a real implementation, we would analyze the WHERE clause to check for unique key conditions
	return false
}

// hasLimit1 checks if a SELECT statement has LIMIT 1
func hasLimit1(stmt *parsercommon.SelectStatement) bool {
	// Check if the statement has a LIMIT clause
	if stmt.Limit == nil {
		return false
	}

	// Check if the LIMIT value is 1
	// This is a placeholder implementation
	// In a real implementation, we would analyze the LIMIT clause to check if it's 1
	return stmt.Limit.Count == 1
}

// isBulkInsert checks if an INSERT statement is a bulk insert
func isBulkInsert(stmt *parsercommon.InsertIntoStatement) bool {
	// This is a placeholder implementation
	// In a real implementation, we would check if the INSERT statement has multiple VALUES clauses
	return false
}
