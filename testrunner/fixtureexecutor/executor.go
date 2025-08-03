package fixtureexecutor

import (
	"database/sql"
	"fmt"
	"runtime"
	"strings"
	"time"

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
	SQL         string                 // SQL query from document
	Parameters  map[string]any         // Parameters from test case
	Options     *ExecutionOptions
	Transaction *sql.Tx
	Executor    *Executor
}

// Executor handles fixture data insertion and query execution
type Executor struct {
	db      *sql.DB
	dialect string
}

// NewExecutor creates a new fixture executor
func NewExecutor(db *sql.DB, dialect string) *Executor {
	return &Executor{
		db:      db,
		dialect: dialect,
	}
}

// ExecuteTest executes a complete test case within a transaction
func (e *Executor) ExecuteTest(testCase *markdownparser.TestCase, sql string, parameters map[string]any, opts *ExecutionOptions) (*ValidationResult, error) {
	tx, err := e.db.Begin()
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
		return nil, fmt.Errorf("unsupported execution mode: %s", execution.Options.Mode)
	}
}

// executeFixtureOnly executes only fixture insertion
func (e *Executor) executeFixtureOnly(execution *TestExecution) (*ValidationResult, error) {
	if err := e.executeFixtures(execution.Transaction, execution.TestCase.Fixtures); err != nil {
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
			if err := e.validateVerifyResults(verifyResult, execution.TestCase.ExpectedResult); err != nil {
				return nil, fmt.Errorf("verify query validation failed: %w", err)
			}
		}
		
		return verifyResult, nil
	}

	// 4. Validate main query results (existing logic)
	if len(execution.TestCase.ExpectedResult) > 0 {
		// For SELECT queries or DML queries with RETURNING clause, do direct result comparison
		if result.QueryType == SelectQuery || hasReturningClause(execution.SQL) {
			if err := e.validateDirectResults(result, execution.TestCase.ExpectedResult); err != nil {
				return nil, fmt.Errorf("direct result validation failed: %w", err)
			}
		} else {
			// For DML queries without RETURNING, use validation specs
			specs, err := parseValidationSpecs(execution.TestCase.ExpectedResult)
			if err != nil {
				return nil, fmt.Errorf("failed to parse validation specs: %w", err)
			}

			if err := e.validateResult(execution.Transaction, result, specs); err != nil {
				return nil, fmt.Errorf("validation failed: %w", err)
			}
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
	
	// TODO: Replace parameters in SQL query
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
		return nil, fmt.Errorf("unsupported query type")
	}
}

// executeSelectQuery executes a SELECT query and returns the data
func (e *Executor) executeSelectQuery(tx *sql.Tx, sqlQuery string) (*ValidationResult, error) {
	rows, err := tx.Query(sqlQuery)
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
		if err := rows.Scan(valuePtrs...); err != nil {
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
	result, err := tx.Exec(sqlQuery)
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
		if err := e.executeTableFixture(tx, fixture); err != nil {
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
		return fmt.Errorf("unsupported insert strategy: %s", fixture.Strategy)
	}
}

// executeClearInsert truncates the table and inserts data
func (e *Executor) executeClearInsert(tx *sql.Tx, fixture markdownparser.TableFixture) error {
	// Truncate table
	if err := e.truncateTable(tx, fixture.TableName); err != nil {
		return fmt.Errorf("failed to truncate table: %w", err)
	}

	// Insert data
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
		return fmt.Errorf("upsert not supported for dialect: %s", e.dialect)
	}
}

// executeDelete deletes rows that match the dataset's primary keys
func (e *Executor) executeDelete(tx *sql.Tx, fixture markdownparser.TableFixture) error {
	// This requires knowledge of primary keys, which we'll implement later
	return fmt.Errorf("delete strategy not yet implemented")
}

// truncateTable truncates a table based on database dialect
func (e *Executor) truncateTable(tx *sql.Tx, tableName string) error {
	var query string
	switch e.dialect {
	case "postgres":
		query = fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", e.quoteIdentifier(tableName))
	case "mysql":
		query = fmt.Sprintf("TRUNCATE TABLE %s", e.quoteIdentifier(tableName))
	case "sqlite":
		query = fmt.Sprintf("DELETE FROM %s", e.quoteIdentifier(tableName))
	default:
		return fmt.Errorf("truncate not supported for dialect: %s", e.dialect)
	}

	_, err := tx.Exec(query)
	return err
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
	stmt, err := tx.Prepare(query)
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

		if _, err := stmt.Exec(values...); err != nil {
			return fmt.Errorf("failed to insert row: %w", err)
		}
	}

	return nil
}

// executePostgresUpsert implements upsert for PostgreSQL
func (e *Executor) executePostgresUpsert(tx *sql.Tx, fixture markdownparser.TableFixture) error {
	// This is a simplified implementation
	// In practice, you'd need to know the primary key columns
	return fmt.Errorf("postgres upsert not yet implemented")
}

// executeMySQLUpsert implements upsert for MySQL
func (e *Executor) executeMySQLUpsert(tx *sql.Tx, fixture markdownparser.TableFixture) error {
	// This is a simplified implementation
	// In practice, you'd need to know the primary key columns
	return fmt.Errorf("mysql upsert not yet implemented")
}

// executeSQLiteUpsert implements upsert for SQLite
func (e *Executor) executeSQLiteUpsert(tx *sql.Tx, fixture markdownparser.TableFixture) error {
	// This is a simplified implementation
	// In practice, you'd need to know the primary key columns
	return fmt.Errorf("sqlite upsert not yet implemented")
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
	var currentQuery strings.Builder
	var queries []string
	
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
		return fmt.Errorf("expected %d result rows, got %d rows", len(expectedResults), len(result.Data))
	}
	
	for i, expectedRow := range expectedResults {
		actualRow := result.Data[i]
		if err := compareRows(expectedRow, actualRow); err != nil {
			return fmt.Errorf("result row %d mismatch: %w", i, err)
		}
	}
	
	return nil
}
func (e *Executor) validateVerifyResults(result *ValidationResult, expectedResults []map[string]any) error {
	if len(result.Data) != len(expectedResults) {
		return fmt.Errorf("expected %d result rows, got %d rows", len(expectedResults), len(result.Data))
	}
	
	for i, expectedRow := range expectedResults {
		actualRow := result.Data[i]
		if err := compareRows(expectedRow, actualRow); err != nil {
			return fmt.Errorf("verify query result row %d mismatch: %w", i, err)
		}
	}
	
	return nil
}
