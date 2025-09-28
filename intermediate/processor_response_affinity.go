package intermediate

import (
	"regexp"
	"strings"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

// ResponseAffinityDetector detects response affinity based on statement type
type ResponseAffinityDetector struct{}

func (r *ResponseAffinityDetector) Name() string {
	return "ResponseAffinityDetector"
}

func (r *ResponseAffinityDetector) Process(ctx *ProcessingContext) error {
	// Use existing DetermineResponseAffinity function
	affinity := determineResponseAffinity(ctx.Statement, ctx.TableInfo)
	ctx.ResponseAffinity = string(affinity)

	return nil
}

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

// determineResponseAffinity analyzes the statement and determines the response affinity
func determineResponseAffinity(stmt parser.StatementNode, tableInfo map[string]*snapsql.TableInfo) ResponseAffinity {
	// Default affinity is "many" for SELECT statements
	affinity := ResponseAffinityMany

	// Determine affinity based on statement type
	switch stmt.Type() {
	case parser.SELECT_STATEMENT:
		// For SELECT statements, check if it's a single row query
		selectStmt, ok := stmt.(*parser.SelectStatement)
		if ok {
			// Check if it has aggregate functions (COUNT, SUM, AVG, etc.)
			if hasAggregateFunctions(selectStmt) {
				affinity = ResponseAffinityOne
			} else if hasLimit1(selectStmt) {
				// Check if it has LIMIT 1
				affinity = ResponseAffinityOne
			} else if hasUniqueKeyCondition(selectStmt, tableInfo) {
				// Check if it's a single row query (e.g., has a WHERE clause with a primary key)
				affinity = ResponseAffinityOne
			} else {
				affinity = ResponseAffinityMany
			}
		}

	case parser.INSERT_INTO_STATEMENT:
		// For INSERT statements, check if it has a RETURNING clause
		insertStmt, ok := stmt.(*parser.InsertIntoStatement)
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

	case parser.UPDATE_STATEMENT:
		// For UPDATE statements, check if it has a RETURNING clause
		updateStmt, ok := stmt.(*parser.UpdateStatement)
		if ok && updateStmt.Returning != nil {
			// UPDATE with RETURNING: check if it has unique key condition
			if hasUniqueKeyConditionForUpdate(updateStmt, tableInfo) {
				affinity = ResponseAffinityOne
			} else {
				affinity = ResponseAffinityMany
			}
		} else {
			// UPDATE without RETURNING doesn't return rows
			affinity = ResponseAffinityNone
		}

	case parser.DELETE_FROM_STATEMENT:
		// For DELETE statements, check if it has a RETURNING clause
		deleteStmt, ok := stmt.(*parser.DeleteFromStatement)
		if ok && deleteStmt.Returning != nil {
			// DELETE with RETURNING: check if it has unique key condition
			if hasUniqueKeyConditionForDelete(deleteStmt, tableInfo) {
				affinity = ResponseAffinityOne
			} else {
				affinity = ResponseAffinityMany
			}
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
func hasUniqueKeyCondition(stmt *parser.SelectStatement, tableInfo map[string]*snapsql.TableInfo) bool {
	if stmt.Where == nil {
		return false
	}

	// Check if this is a JOIN query
	if hasJoinTables(stmt.From) {
		return hasUniqueKeyConditionForJoin(stmt, tableInfo)
	}

	// Handle single table queries (existing logic)
	return hasUniqueKeyConditionForSingleTable(stmt, tableInfo)
}

// hasJoinTables checks if the FROM clause contains JOIN operations
func hasJoinTables(fromClause *parser.FromClause) bool {
	if fromClause == nil || len(fromClause.Tables) <= 1 {
		return false
	}

	// Check if any table has a join type other than JoinNone
	for _, table := range fromClause.Tables {
		if table.JoinType != parser.JoinNone {
			return true
		}
	}

	return false
}

// hasUniqueKeyConditionForJoin checks unique key condition for JOIN queries
func hasUniqueKeyConditionForJoin(stmt *parser.SelectStatement, tableInfo map[string]*snapsql.TableInfo) bool {
	// Get the driving table (first table in FROM clause)
	drivingTable := getDrivingTable(stmt.From)
	if drivingTable == "" {
		return false
	}

	// Get the driving table alias
	drivingAlias := getDrivingTableAlias(stmt.From)

	// Check if all JOINs are INNER or LEFT OUTER
	if !areJoinsAllowedForOne(stmt.From) {
		return false
	}

	// Check if response fields follow the double underscore pattern correctly
	if !hasValidJoinFieldPatternWithAlias(stmt.Select, drivingTable, drivingAlias) {
		return false
	}

	// Check if driving table's primary key is specified in WHERE clause
	return isDrivingTablePrimaryKeySpecifiedWithAlias(stmt.Where, drivingTable, drivingAlias, tableInfo)
}

// hasUniqueKeyConditionForSingleTable handles single table queries (existing logic)
func hasUniqueKeyConditionForSingleTable(stmt *parser.SelectStatement, tableInfo map[string]*snapsql.TableInfo) bool {
	// Get the main table name from FROM clause
	tableName := getMainTableName(stmt.From)
	if tableName == "" {
		return false
	}

	// Get table information
	table, exists := tableInfo[tableName]
	if !exists {
		return false
	}

	// Get primary key columns
	primaryKeys := getPrimaryKeyColumns(table)
	if len(primaryKeys) == 0 {
		return false
	}

	// Check if WHERE clause contains primary key equality conditions
	whereText := getWhereClauseText(stmt.Where)

	// Check if all primary keys are specified with equality
	return areAllPrimaryKeysInWhere(primaryKeys, whereText)
}

// getDrivingTable returns the name of the driving table (first table in FROM clause)
func getDrivingTable(fromClause *parser.FromClause) string {
	if fromClause == nil || len(fromClause.Tables) == 0 {
		return ""
	}

	// The first table is the driving table
	firstTable := fromClause.Tables[0]
	if firstTable.TableName != "" {
		return firstTable.TableName
	}

	return firstTable.Name
}

// getDrivingTableAlias returns the alias of the driving table
func getDrivingTableAlias(fromClause *parser.FromClause) string {
	if fromClause == nil || len(fromClause.Tables) == 0 {
		return ""
	}

	// The first table is the driving table
	firstTable := fromClause.Tables[0]

	return firstTable.Name // This is the alias if present, otherwise the table name
}

// areJoinsAllowedForOne checks if all JOINs are INNER or LEFT OUTER
func areJoinsAllowedForOne(fromClause *parser.FromClause) bool {
	if fromClause == nil {
		return true
	}

	for _, table := range fromClause.Tables {
		// Allow JoinNone (first table), JoinInner, and JoinLeft
		if table.JoinType != parser.JoinNone &&
			table.JoinType != parser.JoinInner &&
			table.JoinType != parser.JoinLeft {
			return false
		}
	}

	return true
}

// hasValidJoinFieldPatternWithAlias checks if response fields follow the double underscore pattern with alias
func hasValidJoinFieldPatternWithAlias(selectClause *parser.SelectClause, drivingTable string, drivingAlias string) bool {
	if selectClause == nil {
		return false
	}

	hasDrivingTableFields := false

	for _, field := range selectClause.Fields {
		fieldName := getFieldOutputName(field)

		if strings.Contains(fieldName, "__") {
			// This is a joined table field - should have double underscore
			continue
		} else {
			// This should be a driving table field
			// Check if it's from the driving table (either explicitly or implicitly)
			if isFieldFromDrivingTableWithAlias(field, drivingTable, drivingAlias) {
				hasDrivingTableFields = true
			}
		}
	}

	// Must have at least some fields from the driving table without double underscore
	return hasDrivingTableFields
}

// isDrivingTablePrimaryKeySpecifiedWithAlias checks if driving table's primary key is in WHERE clause with alias
func isDrivingTablePrimaryKeySpecifiedWithAlias(whereClause *parser.WhereClause, drivingTable string, drivingAlias string, tableInfo map[string]*snapsql.TableInfo) bool {
	if whereClause == nil {
		return false
	}

	// Get table information for driving table
	table, exists := tableInfo[drivingTable]
	if !exists {
		return false
	}

	// Get primary key columns for driving table
	primaryKeys := getPrimaryKeyColumns(table)
	if len(primaryKeys) == 0 {
		return false
	}

	// Get WHERE clause text
	whereText := getWhereClauseText(whereClause)

	// Check if all primary keys of driving table are specified with equality
	// For JOIN queries, we need to check for table-qualified field names with alias
	return areAllPrimaryKeysInWhereForTableWithAlias(primaryKeys, whereText, drivingTable, drivingAlias)
}

// areAllPrimaryKeysInWhereForTableWithAlias checks if all primary keys are specified with equality for a specific table with alias
func areAllPrimaryKeysInWhereForTableWithAlias(primaryKeys []string, whereText string, tableName string, tableAlias string) bool {
	lowerText := strings.ToLower(whereText)
	lowerTableName := strings.ToLower(tableName)
	lowerTableAlias := strings.ToLower(tableAlias)

	for _, key := range primaryKeys {
		lowerKey := strings.ToLower(key)

		// Regex patterns to avoid substring false positives
		tableQualifiedRe := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(lowerTableName) + `\.` + wordBoundary(lowerKey) + `\s*=`)

		var aliasQualifiedRe *regexp.Regexp
		if lowerTableAlias != "" {
			aliasQualifiedRe = regexp.MustCompile(`(?i)` + regexp.QuoteMeta(lowerTableAlias) + `\.` + wordBoundary(lowerKey) + `\s*=`)
		}

		unqualifiedRe := regexp.MustCompile(`(?i)` + wordBoundary(lowerKey) + `\s*=`)

		hasTableMatch := tableQualifiedRe.MatchString(lowerText)
		hasAliasMatch := aliasQualifiedRe != nil && aliasQualifiedRe.MatchString(lowerText)
		hasUnqualifiedMatch := unqualifiedRe.MatchString(lowerText)

		if !hasTableMatch && !hasAliasMatch && !hasUnqualifiedMatch {
			return false
		}
	}

	return true
}

// getFieldOutputName returns the output name of a field (alias if present, otherwise original name)
func getFieldOutputName(field parser.SelectField) string {
	if field.ExplicitName && field.FieldName != "" {
		return field.FieldName
	}

	return field.OriginalField
}

// isFieldFromDrivingTableWithAlias checks if a field belongs to the driving table considering alias
func isFieldFromDrivingTableWithAlias(field parser.SelectField, drivingTable string, drivingAlias string) bool {
	// If field has explicit table name, check if it matches driving table or its alias
	if field.TableName != "" {
		// Check if it's the table name itself
		if field.TableName == drivingTable {
			return true
		}

		// Check if it's the alias
		if field.TableName == drivingAlias {
			return true
		}
	}

	// If no explicit table name, assume it's from driving table for simple fields
	return field.TableName == ""
}

// getMainTableName extracts the main table name from FROM clause
func getMainTableName(fromClause *parser.FromClause) string {
	if fromClause == nil || len(fromClause.Tables) == 0 {
		return ""
	}

	// Get the first table name (main table)
	firstTable := fromClause.Tables[0]
	if firstTable.TableName != "" {
		return firstTable.TableName
	}

	if firstTable.Name != "" {
		return firstTable.Name
	}

	return ""
}

// getPrimaryKeyColumns returns the list of primary key column names
func getPrimaryKeyColumns(table *snapsql.TableInfo) []string {
	var primaryKeys []string

	for _, column := range table.Columns {
		if column.IsPrimaryKey {
			primaryKeys = append(primaryKeys, column.Name)
		}
	}

	return primaryKeys
}

// areAllPrimaryKeysInWhere checks if all primary keys are specified in WHERE clause with equality
func areAllPrimaryKeysInWhere(primaryKeys []string, whereText string) bool {
	lowerText := strings.ToLower(whereText)
	for _, pk := range primaryKeys {
		lowerPK := strings.ToLower(pk)

		re := regexp.MustCompile(`(?i)` + wordBoundary(lowerPK) + `\s*=`)
		if !re.MatchString(lowerText) {
			return false
		}
	}

	return true
}

// wordBoundary builds a regex fragment to match identifier with word boundaries
func wordBoundary(ident string) string {
	// Use \b word boundary which works with RE2. This is sufficient for typical identifiers.
	return `\b` + regexp.QuoteMeta(ident) + `\b`
}

// getWhereClauseText extracts the text content of WHERE clause
func getWhereClauseText(whereClause *parser.WhereClause) string {
	if whereClause == nil {
		return ""
	}

	// Get the source text from the clause
	sourceText := whereClause.SourceText()

	// If source text is just "WHERE", try to get more detailed information
	if sourceText == "WHERE" {
		// Try to get the raw tokens and reconstruct the text
		tokens := whereClause.RawTokens()

		var parts []string

		for _, token := range tokens {
			if token.Type != tok.WHITESPACE {
				parts = append(parts, token.Value)
			}
		}

		if len(parts) > 1 { // Skip the "WHERE" token itself
			reconstructed := strings.Join(parts[1:], " ")
			if reconstructed != "" {
				return reconstructed
			}
		}
	}

	return sourceText
}

// hasLimit1 checks if a SELECT statement has LIMIT 1
func hasLimit1(stmt *parser.SelectStatement) bool {
	// Check if the statement has a LIMIT clause
	if stmt.Limit == nil {
		return false
	}

	// Check if the limit value is 1
	// This is a simplified check - in a real implementation,
	// we would need to evaluate the limit expression
	return stmt.Limit.Count == 1
}

// isBulkInsert checks if an INSERT statement is a bulk insert
func isBulkInsert(stmt *parser.InsertIntoStatement) bool {
	// This is a placeholder implementation
	// In a real implementation, we would check if the INSERT has multiple VALUES clauses
	return false
}

// hasAggregateFunctions checks if a SELECT statement contains aggregate functions
func hasAggregateFunctions(stmt *parser.SelectStatement) bool {
	if stmt.Select == nil {
		return false
	}

	// List of common aggregate functions
	aggregateFunctions := []string{
		"COUNT", "SUM", "AVG", "MIN", "MAX",
		"STDDEV", "VARIANCE", "GROUP_CONCAT", "STRING_AGG",
	}

	// Check each selected field
	for _, field := range stmt.Select.Fields {
		// Check both OriginalField and FieldName
		fieldsToCheck := []string{
			strings.ToUpper(strings.TrimSpace(field.OriginalField)),
			strings.ToUpper(strings.TrimSpace(field.FieldName)),
		}

		for _, fieldText := range fieldsToCheck {
			if fieldText == "" {
				continue
			}

			// Check if the field contains any aggregate function
			for _, aggFunc := range aggregateFunctions {
				// Check for function name followed by optional spaces and opening parenthesis
				pattern := aggFunc + "("

				spacePattern := aggFunc + " ("
				if strings.Contains(fieldText, pattern) || strings.Contains(fieldText, spacePattern) {
					return true
				}
			}
		}
	}

	// Check the body tokens as a fallback
	// This handles cases where the parser doesn't properly extract field information
	if bodyTokens := stmt.Select.ContentTokens(); len(bodyTokens) > 0 {
		var bodyText strings.Builder
		for _, token := range bodyTokens {
			bodyText.WriteString(token.Value)
			bodyText.WriteString(" ")
		}

		selectText := strings.ToUpper(bodyText.String())

		for _, aggFunc := range aggregateFunctions {
			// Check for function name followed by optional spaces and opening parenthesis
			pattern := aggFunc + "("

			spacePattern := aggFunc + " ("
			if strings.Contains(selectText, pattern) || strings.Contains(selectText, spacePattern) {
				return true
			}
		}
	}

	return false
}

// hasUniqueKeyConditionForUpdate checks if an UPDATE statement has a WHERE clause with a unique key condition
func hasUniqueKeyConditionForUpdate(stmt *parser.UpdateStatement, tableInfo map[string]*snapsql.TableInfo) bool {
	if stmt.Where == nil {
		return false
	}

	// Get the table name from UPDATE clause
	tableName := getUpdateTableName(stmt)
	if tableName == "" {
		return false
	}

	// Get table info
	info, exists := tableInfo[tableName]
	if !exists {
		return false
	}

	// Get primary key columns
	primaryKeys := getPrimaryKeyColumns(info)
	if len(primaryKeys) == 0 {
		return false
	}

	// Check if WHERE clause contains all primary key conditions
	return checkPrimaryKeyConditions(stmt.Where, primaryKeys, tableName, "")
}

// hasUniqueKeyConditionForDelete checks if a DELETE statement has a WHERE clause with a unique key condition
func hasUniqueKeyConditionForDelete(stmt *parser.DeleteFromStatement, tableInfo map[string]*snapsql.TableInfo) bool {
	if stmt.Where == nil {
		return false
	}

	// Get the table name from DELETE clause
	tableName := getDeleteTableName(stmt)
	if tableName == "" {
		return false
	}

	// Get table info
	info, exists := tableInfo[tableName]
	if !exists {
		return false
	}

	// Get primary key columns
	primaryKeys := getPrimaryKeyColumns(info)
	if len(primaryKeys) == 0 {
		return false
	}

	// Check if WHERE clause contains all primary key conditions
	return checkPrimaryKeyConditions(stmt.Where, primaryKeys, tableName, "")
}

// getUpdateTableName extracts the table name from UPDATE statement
func getUpdateTableName(stmt *parser.UpdateStatement) string {
	if stmt.Update == nil {
		return ""
	}

	return stmt.Update.Table.TableName
}

// getDeleteTableName extracts the table name from DELETE statement
func getDeleteTableName(stmt *parser.DeleteFromStatement) string {
	if stmt.From == nil {
		return ""
	}

	return stmt.From.Table.TableName
}

// checkPrimaryKeyConditions checks if WHERE clause contains all primary key conditions
func checkPrimaryKeyConditions(whereClause *parser.WhereClause, primaryKeys []string, tableName string, tableAlias string) bool {
	if whereClause == nil || len(primaryKeys) == 0 {
		return false
	}

	// Get WHERE clause text
	whereText := getWhereClauseText(whereClause)

	// Check if all primary keys are specified with equality
	if tableAlias != "" {
		return areAllPrimaryKeysInWhereForTableWithAlias(primaryKeys, whereText, tableName, tableAlias)
	} else {
		return areAllPrimaryKeysInWhere(primaryKeys, whereText)
	}
}
