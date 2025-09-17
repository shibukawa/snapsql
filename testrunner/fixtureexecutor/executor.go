package fixtureexecutor

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/markdownparser"
)

// ExecutionMode represents the test execution mode
type ExecutionMode string

const (
	FullTest    ExecutionMode = "full-test"
	FixtureOnly ExecutionMode = "fixture-only"
	QueryOnly   ExecutionMode = "query-only"
)

// ExecutionOptions contains options for test execution
type ExecutionOptions struct {
	Mode     ExecutionMode
	Commit   bool
	Parallel int
	Timeout  time.Duration
}

// DefaultExecutionOptions returns default execution options
func DefaultExecutionOptions() *ExecutionOptions {
	return &ExecutionOptions{
		Mode:     FullTest,
		Commit:   false,
		Parallel: runtime.NumCPU(),
		Timeout:  time.Minute * 10,
	}
}

// QueryType represents the type of SQL query
type QueryType int

const (
	SelectQuery QueryType = iota
	InsertQuery
	UpdateQuery
	DeleteQuery
)

// ValidationResult contains the result of query execution and validation
type ValidationResult struct {
	Data         []map[string]any
	RowsAffected int64
	QueryType    QueryType
}

// TestExecution represents a single test execution context
type TestExecution struct {
	TestCase    *markdownparser.TestCase
	SQL         string         // SQL query from document
	Parameters  map[string]any // Parameters from test case
	Options     *ExecutionOptions
	Transaction *sql.Tx
	Executor    *Executor
}

// Executor handles fixture data insertion and query execution
type Executor struct {
	db        *sql.DB
	dialect   string
	tableInfo map[string]*snapsql.TableInfo
}

// NewExecutor creates a new fixture executor
func NewExecutor(db *sql.DB, dialect string, tableInfo map[string]*snapsql.TableInfo) *Executor {
	return &Executor{
		db:        db,
		dialect:   dialect,
		tableInfo: tableInfo,
	}
}

// ExecuteTest executes a complete test case within a transaction
func (e *Executor) ExecuteTest(testCase *markdownparser.TestCase, sql string, parameters map[string]any, opts *ExecutionOptions) (*ValidationResult, error) {
	ctx := context.Background()

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if opts.Commit {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	execution := &TestExecution{
		TestCase:    testCase,
		SQL:         sql,
		Parameters:  parameters,
		Options:     opts,
		Transaction: tx,
		Executor:    e,
	}

	return e.executeTestSteps(execution)
}

// executeTestSteps executes the test steps based on execution mode
func (e *Executor) executeTestSteps(execution *TestExecution) (*ValidationResult, error) {
	switch execution.Options.Mode {
	case FixtureOnly:
		return e.executeFixtureOnly(execution)
	case QueryOnly:
		return e.executeQueryOnly(execution)
	case FullTest:
		return e.executeFullTest(execution)
	default:
		return nil, fmt.Errorf("%w: %s", snapsql.ErrUnsupportedExecutionMode, execution.Options.Mode)
	}
}

// executeFixtureOnly executes only fixture insertion
func (e *Executor) executeFixtureOnly(execution *TestExecution) (*ValidationResult, error) {
	err := e.executeFixtures(execution.Transaction, execution.TestCase.Fixtures)
	if err != nil {
		return nil, err
	}

	return &ValidationResult{
		Data:         []map[string]any{{"status": "fixtures_inserted"}},
		RowsAffected: int64(len(execution.TestCase.Fixtures)),
		QueryType:    InsertQuery,
	}, nil
}

// executeQueryOnly executes only the query without fixtures
func (e *Executor) executeQueryOnly(execution *TestExecution) (*ValidationResult, error) {
	// Execute the SQL query
	result, err := e.executeQuery(execution.Transaction, execution.SQL, execution.Parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	// Validate results if expected results are provided
	if len(execution.TestCase.ExpectedResult) > 0 {
		specs, err := parseValidationSpecs(execution.TestCase.ExpectedResult)
		if err != nil {
			return nil, fmt.Errorf("failed to parse validation specs: %w", err)
		}

		if err := e.validateResult(execution.Transaction, result, specs); err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}
	}

	return result, nil
}

// executeFullTest executes the complete test flow
func (e *Executor) executeFullTest(execution *TestExecution) (*ValidationResult, error) {
	// 1. Execute fixtures
	if err := e.executeFixtures(execution.Transaction, execution.TestCase.Fixtures); err != nil {
		return nil, fmt.Errorf("failed to execute fixtures: %w", err)
	}

	// 2. Execute main query
	result, err := e.executeQuery(execution.Transaction, execution.SQL, execution.Parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	// 3. Execute verify query if present
	if execution.TestCase.VerifyQuery != "" {
		verifyResult, err := e.executeVerifyQuery(execution.Transaction, execution.TestCase.VerifyQuery)
		if err != nil {
			return nil, fmt.Errorf("failed to execute verify query: %w", err)
		}

		// 4. Validate verify query results
		if len(execution.TestCase.ExpectedResult) > 0 {
			err := e.validateVerifyResults(verifyResult, execution.TestCase.ExpectedResult)
			if err != nil {
				return nil, fmt.Errorf("verify query validation failed: %w", err)
			}
		}

		return verifyResult, nil
	}

	// 4. Validate (暫定: 旧式 ExpectedResult を直接比較)
	if len(execution.TestCase.ExpectedResult) > 0 && (result.QueryType == SelectQuery || hasReturningClause(execution.SQL)) {
		if err := compareRowsSlice(execution.TestCase.ExpectedResult, result.Data); err != nil {
			return nil, fmt.Errorf("simple validation failed: %w", err)
		}
	}

	return result, nil
}

// detectQueryType detects the type of SQL query
func detectQueryType(sql string) QueryType {
	// Remove leading whitespace and convert to uppercase
	trimmed := strings.TrimSpace(strings.ToUpper(sql))

	if strings.HasPrefix(trimmed, "SELECT") || strings.HasPrefix(trimmed, "WITH") {
		return SelectQuery
	} else if strings.HasPrefix(trimmed, "INSERT") {
		return InsertQuery
	} else if strings.HasPrefix(trimmed, "UPDATE") {
		return UpdateQuery
	} else if strings.HasPrefix(trimmed, "DELETE") {
		return DeleteQuery
	}

	// Default to SELECT for unknown queries
	return SelectQuery
}

// hasReturningClause checks if the SQL query has a RETURNING clause
func hasReturningClause(sql string) bool {
	// Convert to uppercase and check for RETURNING keyword
	upperSQL := strings.ToUpper(sql)
	return strings.Contains(upperSQL, "RETURNING")
}

// executeQuery executes the SQL query and returns the result
func (e *Executor) executeQuery(tx *sql.Tx, sqlQuery string, parameters map[string]any) (*ValidationResult, error) {
	queryType := detectQueryType(sqlQuery)

	// Parameter replacement in SQL query is handled by the template engine
	// For now, execute the query as-is

	// Check for RETURNING clause in DML queries
	if (queryType == InsertQuery || queryType == UpdateQuery || queryType == DeleteQuery) && hasReturningClause(sqlQuery) {
		// Execute as SELECT query to get returned data
		result, err := e.executeSelectQuery(tx, sqlQuery)
		if err != nil {
			return nil, err
		}
		// Keep the original query type for validation logic
		result.QueryType = queryType

		return result, nil
	}

	switch queryType {
	case SelectQuery:
		return e.executeSelectQuery(tx, sqlQuery)
	case InsertQuery, UpdateQuery, DeleteQuery:
		return e.executeDMLQuery(tx, sqlQuery, queryType)
	default:
		return nil, snapsql.ErrUnsupportedQueryType
	}
}

// executeSelectQuery executes a SELECT query and returns the data
func (e *Executor) executeSelectQuery(tx *sql.Tx, sqlQuery string) (*ValidationResult, error) {
	ctx := context.Background()

	rows, err := tx.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SELECT query: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get column names: %w", err)
	}

	// Prepare result data
	var data []map[string]any

	for rows.Next() {
		// Create slice of interface{} for scanning
		values := make([]interface{}, len(columns))

		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan the row
		err := rows.Scan(valuePtrs...)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert to map
		row := make(map[string]any)

		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				// Convert byte slices to strings
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}

		data = append(data, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return &ValidationResult{
		Data:         data,
		RowsAffected: int64(len(data)),
		QueryType:    SelectQuery,
	}, nil
}

// executeDMLQuery executes INSERT/UPDATE/DELETE queries and returns affected rows
func (e *Executor) executeDMLQuery(tx *sql.Tx, sqlQuery string, queryType QueryType) (*ValidationResult, error) {
	ctx := context.Background()

	result, err := tx.ExecContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to execute DML query: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return &ValidationResult{
		Data:         []map[string]any{{"rows_affected": rowsAffected}},
		RowsAffected: rowsAffected,
		QueryType:    queryType,
	}, nil
}
func (e *Executor) executeFixtures(tx *sql.Tx, fixtures []markdownparser.TableFixture) error {
	for _, fixture := range fixtures {
		err := e.executeTableFixture(tx, fixture)
		if err != nil {
			return fmt.Errorf("failed to execute fixture for table %s: %w", fixture.TableName, err)
		}
	}

	return nil
}

// executeTableFixture executes a single table fixture based on its strategy
func (e *Executor) executeTableFixture(tx *sql.Tx, fixture markdownparser.TableFixture) error {
	switch fixture.Strategy {
	case markdownparser.ClearInsert:
		return e.executeClearInsert(tx, fixture)
	case markdownparser.Insert:
		return e.executeInsert(tx, fixture)
	case markdownparser.Upsert:
		return e.executeUpsert(tx, fixture)
	case markdownparser.Delete:
		return e.executeDelete(tx, fixture)
	default:
		return fmt.Errorf("%w: %s", snapsql.ErrUnsupportedInsertStrategy, fixture.Strategy)
	}
}

// executeClearInsert truncates the table and inserts data
func (e *Executor) executeClearInsert(tx *sql.Tx, fixture markdownparser.TableFixture) error {
	// 簡易DELETE実装（dialect依存truncateは未実装暫定）
	query := fmt.Sprintf("DELETE FROM %s", e.quoteIdentifier(fixture.TableName))
	if _, err := tx.ExecContext(context.Background(), query); err != nil {
		return fmt.Errorf("failed to clear table %s: %w", fixture.TableName, err)
	}
	return e.insertData(tx, fixture.TableName, fixture.Data)
}

// executeInsert just inserts data into the table
func (e *Executor) executeInsert(tx *sql.Tx, fixture markdownparser.TableFixture) error {
	return e.insertData(tx, fixture.TableName, fixture.Data)
}

// executeUpsert inserts data or updates if exists
func (e *Executor) executeUpsert(tx *sql.Tx, fixture markdownparser.TableFixture) error {
	// Implementation depends on database dialect
	switch e.dialect {
	case "postgres":
		return e.executePostgresUpsert(tx, fixture)
	case "mysql":
		return e.executeMySQLUpsert(tx, fixture)
	case "sqlite":
		return e.executeSQLiteUpsert(tx, fixture)
	default:
		return fmt.Errorf("%w: %s", snapsql.ErrUpsertNotSupported, e.dialect)
	}
}

// executeDelete deletes rows that match the dataset's primary keys
func (e *Executor) executeDelete(tx *sql.Tx, fixture markdownparser.TableFixture) error {
	if len(fixture.Data) == 0 {
		return nil
	}

	// 主キー情報取得
	tblInfo, ok := e.tableInfo[fixture.TableName]
	if !ok || tblInfo == nil {
		return fmt.Errorf("table info not found for table: %s", fixture.TableName)
	}
	var pkCols []string
	for colName, colInfo := range tblInfo.Columns {
		if colInfo.IsPrimaryKey {
			pkCols = append(pkCols, colName)
		}
	}
	if len(pkCols) == 0 {
		return fmt.Errorf("no primary key defined for table: %s", fixture.TableName)
	}

	for _, row := range fixture.Data {
		var (
			whereClauses []string
			values       []any
			idx          = 1
		)
		for _, pk := range pkCols {
			val, exists := row[pk]
			if !exists {
				return fmt.Errorf("primary key column %s missing in fixture data for table %s", pk, fixture.TableName)
			}
			whereClauses = append(whereClauses, fmt.Sprintf("%s = %s", e.quoteIdentifier(pk), e.getPlaceholder(idx)))
			values = append(values, val)
			idx++
		}
		// 主キー以外のカラムは無視
		query := fmt.Sprintf("DELETE FROM %s WHERE %s", e.quoteIdentifier(fixture.TableName), strings.Join(whereClauses, " AND "))
		ctx := context.Background()
		if _, err := tx.ExecContext(ctx, query, values...); err != nil {
			return fmt.Errorf("failed to execute delete for table %s: %w", fixture.TableName, err)
		}
	}
	return nil
}

// getPrimaryKeyColumns: テーブルの主キー列名リストを返す
func (e *Executor) getPrimaryKeyColumns(tableName string) ([]string, error) {
	tblInfo, ok := e.tableInfo[tableName]
	if !ok || tblInfo == nil {
		return nil, fmt.Errorf("table info not found for table: %s", tableName)
	}
	var pkCols []string
	for col, info := range tblInfo.Columns {
		if info.IsPrimaryKey {
			pkCols = append(pkCols, col)
		}
	}
	if len(pkCols) == 0 {
		return nil, fmt.Errorf("no primary key defined for table: %s", tableName)
	}
	return pkCols, nil
}

// matchPrimaryKey: 主キー列で2行が一致するか
func matchPrimaryKey(pkCols []string, row1, row2 map[string]any) bool {
	for _, pk := range pkCols {
		v1, ok1 := row1[pk]
		v2, ok2 := row2[pk]
		if !ok1 || !ok2 {
			return false
		}
		if v1 != v2 {
			return false
		}
	}
	return true
}

// extractPrimaryKey: 主キー値のみ抽出
func extractPrimaryKey(pkCols []string, row map[string]any) map[string]any {
	m := make(map[string]any)
	for _, pk := range pkCols {
		m[pk] = row[pk]
	}
	return m
}

// compareRowsSlice: 2つの[]map[string]anyを順序・件数・値で完全一致比較
func compareRowsSlice(expected, actual []map[string]any) error {
	if len(expected) != len(actual) {
		return fmt.Errorf("row count mismatch: expected %d, got %d", len(expected), len(actual))
	}
	for i := range expected {
		if err := compareRowsWithMatchers(expected[i], actual[i]); err != nil {
			return fmt.Errorf("row %d: %w", i, err)
		}
	}
	return nil
}

// compareRowsWithMatchers: 1行分の値比較（値比較特殊指定対応）
func compareRowsWithMatchers(expected, actual map[string]any) error {
	for k, vExp := range expected {
		vAct, ok := actual[k]
		if !ok {
			return fmt.Errorf("column %s missing in actual row", k)
		}
		// 値比較特殊指定
		switch val := vExp.(type) {
		case []any:
			// [null], [notnull], [any], [regexp, ...]
			if len(val) == 1 {
				switch val[0] {
				case "null":
					if vAct != nil {
						return fmt.Errorf("column %s: expected null, got %v", k, vAct)
					}
				case "notnull":
					if vAct == nil {
						return fmt.Errorf("column %s: expected notnull, got null", k)
					}
				case "any":
					// 何でもOK
				default:
					return fmt.Errorf("column %s: unknown matcher %v", k, val[0])
				}
			} else if len(val) == 2 && val[0] == "regexp" {
				pat, ok := val[1].(string)
				if !ok {
					return fmt.Errorf("column %s: regexp pattern must be string", k)
				}
				s, ok := vAct.(string)
				if !ok {
					return fmt.Errorf("column %s: regexp matcher expects string, got %T", k, vAct)
				}
				matched, err := regexp.MatchString(pat, s)
				if err != nil {
					return fmt.Errorf("column %s: regexp error: %w", k, err)
				}
				if !matched {
					return fmt.Errorf("column %s: value '%s' does not match regexp '%s'", k, s, pat)
				}
			} else {
				return fmt.Errorf("column %s: invalid matcher syntax: %v", k, val)
			}
		default:
			// 通常値比較
			if !valueEquals(vExp, vAct) {
				return fmt.Errorf("column %s: expected %v, got %v", k, vExp, vAct)
			}
		}
	}
	return nil
}

// valueEquals: 厳密一致（float/int/文字列/その他）
func valueEquals(a, b any) bool {
	// nil 同士
	if a == nil || b == nil {
		return a == b
	}

	// string と []byte の比較（SQLite で TEXT が []byte になるケース緩和）
	if sa, ok := a.(string); ok {
		if bb, ok2 := b.([]byte); ok2 {
			return sa == string(bb)
		}
	}
	if sb, ok := b.(string); ok {
		if ab, ok2 := a.([]byte); ok2 {
			return string(ab) == sb
		}
	}

	// 数値型の包括比較
	if fa, ok := toFloat(a); ok {
		if fb, ok2 := toFloat(b); ok2 {
			// ここは整数同士なら誤差不要だが一律で扱う
			if fa == fb {
				return true
			}
			// 浮動小数点誤差吸収（将来拡張用）
			if math.Abs(fa-fb) < 1e-9 {
				return true
			}
			return false
		}
	}

	return a == b
}

// toFloat: 任意の数値型を float64 に正規化
func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}

// insertData inserts data into a table
func (e *Executor) insertData(tx *sql.Tx, tableName string, data []map[string]any) error {
	if len(data) == 0 {
		return nil
	}

	// Get column names from the first row
	var columns []string
	for col := range data[0] {
		columns = append(columns, col)
	}

	// Build INSERT query
	quotedColumns := make([]string, len(columns))
	for i, col := range columns {
		quotedColumns[i] = e.quoteIdentifier(col)
	}

	placeholders := make([]string, len(columns))
	for i := range placeholders {
		placeholders[i] = e.getPlaceholder(i + 1)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		e.quoteIdentifier(tableName),
		strings.Join(quotedColumns, ", "),
		strings.Join(placeholders, ", "))

	// Prepare statement
	ctx := context.Background()

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	// Insert each row
	for _, row := range data {
		values := make([]any, len(columns))
		for i, col := range columns {
			values[i] = row[col]
		}

		if _, err := stmt.ExecContext(ctx, values...); err != nil {
			return fmt.Errorf("failed to insert row: %w", err)
		}
	}

	return nil
}

// executePostgresUpsert implements upsert for PostgreSQL
func (e *Executor) executePostgresUpsert(tx *sql.Tx, fixture markdownparser.TableFixture) error {
	// This is a simplified implementation
	// In practice, you'd need to know the primary key columns
	return snapsql.ErrPostgresUpsertNotImplemented
}

// executeMySQLUpsert implements upsert for MySQL
func (e *Executor) executeMySQLUpsert(tx *sql.Tx, fixture markdownparser.TableFixture) error {
	// This is a simplified implementation
	// In practice, you'd need to know the primary key columns
	return snapsql.ErrMysqlUpsertNotImplemented
}

// executeSQLiteUpsert implements upsert for SQLite
func (e *Executor) executeSQLiteUpsert(tx *sql.Tx, fixture markdownparser.TableFixture) error {
	// This is a simplified implementation
	// In practice, you'd need to know the primary key columns
	return snapsql.ErrSqliteUpsertNotImplemented
}

// quoteIdentifier quotes database identifiers based on dialect
func (e *Executor) quoteIdentifier(identifier string) string {
	switch e.dialect {
	case "postgres":
		return fmt.Sprintf(`"%s"`, identifier)
	case "mysql":
		return fmt.Sprintf("`%s`", identifier)
	case "sqlite":
		return fmt.Sprintf(`"%s"`, identifier)
	default:
		return identifier
	}
}

// getPlaceholder returns the appropriate placeholder for the dialect
func (e *Executor) getPlaceholder(position int) string {
	switch e.dialect {
	case "postgres":
		return fmt.Sprintf("$%d", position)
	case "mysql", "sqlite":
		return "?"
	default:
		return "?"
	}
}

// executeVerifyQuery executes the verify query and returns the result
func (e *Executor) executeVerifyQuery(tx *sql.Tx, verifyQuery string) (*ValidationResult, error) {
	// Split multiple queries by semicolon
	queries := e.parseMultipleQueries(verifyQuery)

	var allResults []map[string]any

	for _, query := range queries {
		if strings.TrimSpace(query) == "" {
			continue
		}

		result, err := e.executeSelectQuery(tx, query)
		if err != nil {
			return nil, fmt.Errorf("failed to execute verify query: %w", err)
		}

		allResults = append(allResults, result.Data...)
	}

	return &ValidationResult{
		Data:         allResults,
		RowsAffected: int64(len(allResults)),
		QueryType:    SelectQuery,
	}, nil
}

// parseMultipleQueries splits SQL string into individual queries
func (e *Executor) parseMultipleQueries(sql string) []string {
	lines := strings.Split(sql, "\n")

	var (
		currentQuery strings.Builder
		queries      []string
	)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip comment-only lines
		if strings.HasPrefix(trimmed, "--") {
			continue
		}

		currentQuery.WriteString(line)
		currentQuery.WriteString("\n")

		// Check for query termination
		if strings.HasSuffix(trimmed, ";") {
			query := strings.TrimSpace(currentQuery.String())
			if query != "" {
				queries = append(queries, query)
			}

			currentQuery.Reset()
		}
	}

	// Add remaining query if any
	if currentQuery.Len() > 0 {
		query := strings.TrimSpace(currentQuery.String())
		if query != "" {
			queries = append(queries, query)
		}
	}

	return queries
}

// validateDirectResults validates direct query results for SELECT queries
func (e *Executor) validateDirectResults(result *ValidationResult, expectedResults []map[string]any) error {
	if len(result.Data) != len(expectedResults) {
		return fmt.Errorf("%w: expected %d result rows, got %d rows", snapsql.ErrResultRowCountMismatch, len(expectedResults), len(result.Data))
	}

	for i, expectedRow := range expectedResults {
		actualRow := result.Data[i]

		err := compareRowsBasic(expectedRow, actualRow)
		if err != nil {
			return fmt.Errorf("result row %d mismatch: %w", i, err)
		}
	}

	return nil
}
func (e *Executor) validateVerifyResults(result *ValidationResult, expectedResults []map[string]any) error {
	if len(result.Data) != len(expectedResults) {
		return fmt.Errorf("%w: expected %d result rows, got %d rows", snapsql.ErrResultRowCountMismatch, len(expectedResults), len(result.Data))
	}

	for i, expectedRow := range expectedResults {
		actualRow := result.Data[i]

		err := compareRowsBasic(expectedRow, actualRow)
		if err != nil {
			return fmt.Errorf("verify query result row %d mismatch: %w", i, err)
		}
	}

	return nil
}
