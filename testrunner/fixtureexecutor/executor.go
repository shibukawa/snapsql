//nolint:all // Temporarily disable linters for this file due to generator-like structure and strict whitespace rules; revisit and re-enable selectively later.
package fixtureexecutor

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/markdownparser"
)

var (
	errTableInfoNotFound     = errors.New("table info not found")
	errNoPrimaryKeyDefined   = errors.New("no primary key defined")
	errPrimaryKeyColumnMiss  = errors.New("primary key column missing in fixture data")
	errRowCountMismatch      = errors.New("row count mismatch")
	errColumnMissing         = errors.New("column missing in actual row")
	errExpectedNull          = errors.New("expected null")
	errExpectedNotNull       = errors.New("expected notnull but got null")
	errUnknownMatcher        = errors.New("unknown matcher")
	errRegexpPatternType     = errors.New("regexp pattern must be string")
	errRegexpExpectString    = errors.New("regexp matcher expects string")
	errRegexpNotMatch        = errors.New("regexp not matched")
	errInvalidMatcherSyntax  = errors.New("invalid matcher syntax")
	errValueMismatch         = errors.New("value mismatch")
	errUpsertMissingPK       = errors.New("upsert row missing primary key column")
	errMissingRequiredColumn = errors.New("missing required non-null column in fixture row")
	errUnknownFixtureColumn  = errors.New("fixture row contains unknown column")
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
		Timeout:  2 * time.Minute,
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

// ExpectedResultsStrategy defines comparison strategy for table state validation.
// Recognized values (design doc): "all" (default), "pk-match", "pk-exists", "pk-not-exists".
// Executor treats empty string as "all" for backward compatibility.

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
	baseDir   string
}

// NewExecutor creates a new fixture executor
func NewExecutor(db *sql.DB, dialect string, tableInfo map[string]*snapsql.TableInfo) *Executor {
	return &Executor{
		db:        db,
		dialect:   dialect,
		tableInfo: tableInfo,
	}
}

// SetBaseDir sets the base directory used to resolve relative external file references
func (e *Executor) SetBaseDir(dir string) { e.baseDir = dir }

// ExecuteTest executes a complete test case within a transaction
func (e *Executor) ExecuteTest(testCase *markdownparser.TestCase, sql string, parameters map[string]any, opts *ExecutionOptions) (*ValidationResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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

// validateTableState applies expected results strategies (table-qualified expected specs)
// to a snapshot of the current table data (queried via verify query or future table fetch logic).
// NOTE: For now we only support strategies against execution.TestCase.ExpectedResults when
// a table name is provided and the original (legacy) ExpectedResult slice is empty.
// SELECT/RETURNING queries still use validateVerifyResults.
func (e *Executor) validateTableStateBySpec(tx *sql.Tx, spec markdownparser.ExpectedResultSpec) error {
	if spec.TableName == "" {
		return nil // Nothing to do (legacy path handles unnamed expected results)
	}
	strategy := spec.Strategy
	if strategy == "" {
		strategy = "all"
	}

	// Load expected data from external file if specified
	if spec.ExternalFile != "" && len(spec.Data) == 0 {
		rows, err := e.loadExternalRows(spec.ExternalFile)
		if err != nil {
			return fmt.Errorf("failed to load expected results from external file: %w", err)
		}
		spec.Data = rows
	}

	// Load tableInfo (need primary key for pk-* strategies)
	ti, ok := e.tableInfo[spec.TableName]
	if !ok {
		return fmt.Errorf("%w: %s", errTableInfoNotFound, spec.TableName)
	}

	// Build SELECT to fetch current table rows (simple full scan ordered by primary keys if exists)
	cols := make([]string, 0, len(ti.Columns))
	for name := range ti.Columns {
		cols = append(cols, name)
	}
	// Deterministic order: order by primary key columns (if any)
	order := ""
	pkCols := make([]string, 0)
	for _, c := range ti.Columns {
		if c.IsPrimaryKey {
			pkCols = append(pkCols, c.Name)
		}
	}
	if len(pkCols) > 0 {
		order = " ORDER BY " + strings.Join(pkCols, ",")
	}
	query := fmt.Sprintf("SELECT %s FROM %s%s", strings.Join(cols, ","), spec.TableName, order)
	// Use a bounded context to avoid indefinite blocking (especially on SQLite under edge cases)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query table state: %w", err)
	}
	defer rows.Close()

	colTypes, _ := rows.ColumnTypes()
	colNames := make([]string, len(colTypes))
	for i, ct := range colTypes {
		colNames[i] = ct.Name()
	}
	actual := make([]map[string]any, 0)
	for rows.Next() {
		scanVals := make([]any, len(colNames))
		scanPtrs := make([]any, len(colNames))
		for i := range scanVals {
			scanPtrs[i] = &scanVals[i]
		}
		if err := rows.Scan(scanPtrs...); err != nil {
			return err
		}
		rowMap := make(map[string]any)
		for i, n := range colNames {
			rowMap[n] = scanVals[i]
		}
		actual = append(actual, rowMap)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	switch strategy {
	case "all":
		// expect full match with order irrelevant? design doc implies exact table contents.
		// We compare counts and then match rows by index after sorting by PK (already ordered if PK exists).
		if err := compareRowsSlice(spec.Data, actual); err != nil {
			return err
		}
		return nil
	case "pk-match":
		return e.comparePKMatch(ti, spec.Data, actual, true)
	case "pk-exists":
		return e.comparePKMatch(ti, spec.Data, actual, false)
	case "pk-not-exists":
		return e.comparePKNotExists(ti, spec.Data, actual)
	default:
		return fmt.Errorf("unknown expected results strategy: %s", strategy)
	}
}

// comparePKMatch: For pk-match requires specified PK rows exist and their non-PK values (provided in expected) match.
// For pk-exists only presence of PK combination is required (other columns ignored).
func (e *Executor) comparePKMatch(ti *snapsql.TableInfo, expected, actual []map[string]any, checkValues bool) error {
	pkCols := make([]string, 0)
	for _, c := range ti.Columns {
		if c.IsPrimaryKey {
			pkCols = append(pkCols, c.Name)
		}
	}
	if len(pkCols) == 0 {
		return errNoPrimaryKeyDefined
	}

	// index actual by PK tuple string
	actualIndex := make(map[string]map[string]any)
	for _, row := range actual {
		actualIndex[pkKey(pkCols, row)] = row
	}

	for i, expRow := range expected {
		key := pkKey(pkCols, expRow)
		actRow, ok := actualIndex[key]
		if !ok {
			return fmt.Errorf("pk row not found (strategy=%v) index=%d key=%s", bool(checkValues), i, key)
		}
		if checkValues {
			// Build expected subset (only columns provided in expRow) and compare with matchers
			if err := compareRowsWithMatchers(expRow, actRow); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *Executor) comparePKNotExists(ti *snapsql.TableInfo, expected, actual []map[string]any) error {
	pkCols := make([]string, 0)
	for _, c := range ti.Columns {
		if c.IsPrimaryKey {
			pkCols = append(pkCols, c.Name)
		}
	}
	if len(pkCols) == 0 {
		return errNoPrimaryKeyDefined
	}

	actualIndex := make(map[string]struct{})
	for _, row := range actual {
		actualIndex[pkKey(pkCols, row)] = struct{}{}
	}
	for i, expRow := range expected {
		key := pkKey(pkCols, expRow)
		if _, exists := actualIndex[key]; exists {
			return fmt.Errorf("pk row unexpectedly exists index=%d key=%s", i, key)
		}
	}
	return nil
}

func pkKey(pkCols []string, row map[string]any) string {
	vals := make([]string, len(pkCols))
	for i, c := range pkCols {
		vals[i] = fmt.Sprintf("%v", row[c])
	}
	return strings.Join(vals, "||")
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

	// 2. Execute main query (only when necessary)
	// If we only validate table state (no verify query, no direct/unnamed expected result rows),
	// and the main SQL is a side-effect-free SELECT (e.g., "SELECT 1"), we can skip executing it.
	// This avoids unnecessary row iteration that can cause timeouts under certain drivers.
	var result *ValidationResult
	queryType := detectQueryType(execution.SQL)
	_, hasUnnamedExternal := firstUnnamedExternalSpec(execution.TestCase.ExpectedResults)
	onlyTableStateCheck := execution.TestCase.VerifyQuery == "" && len(execution.TestCase.ExpectedResult) == 0 && !hasUnnamedExternal
	hasTableQualifiedSpecs := false
	for _, spec := range execution.TestCase.ExpectedResults {
		if spec.TableName != "" {
			hasTableQualifiedSpecs = true
			break
		}
	}
	skipMainSelect := (queryType == SelectQuery) && onlyTableStateCheck && hasTableQualifiedSpecs

	if skipMainSelect {
		// Execute the SQL to honor potential side effects while avoiding row iteration cost.
		// Prefer ExecContext; if driver doesn't allow Exec on SELECT, fall back to QueryContext and close immediately.
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, err := execution.Transaction.ExecContext(ctx, execution.SQL); err != nil {
			// Fallback: run as QueryContext then close immediately without iteration
			if rows, qerr := execution.Transaction.QueryContext(ctx, execution.SQL); qerr == nil {
				rows.Close()
			} else {
				return nil, fmt.Errorf("failed to execute main SQL (exec/query fallback): exec=%v query=%v", err, qerr)
			}
		}
		result = &ValidationResult{Data: nil, RowsAffected: 0, QueryType: SelectQuery}
	} else {
		var err error
		result, err = e.executeQuery(execution.Transaction, execution.SQL, execution.Parameters)
		if err != nil {
			return nil, fmt.Errorf("failed to execute query: %w", err)
		}
	}

	// 3. Execute verify query if present
	if execution.TestCase.VerifyQuery != "" {
		verifyResult, err := e.executeVerifyQuery(execution.Transaction, execution.TestCase.VerifyQuery)
		if err != nil {
			return nil, fmt.Errorf("failed to execute verify query: %w", err)
		}

		// 4. Validate verify query results (legacy unnamed)
		if len(execution.TestCase.ExpectedResult) > 0 {
			if err := e.validateVerifyResults(verifyResult, execution.TestCase.ExpectedResult); err != nil {
				return nil, fmt.Errorf("verify query validation failed: %w", err)
			}
		} else {
			// Support unnamed external expected results via ExpectedResults entry with empty TableName
			if spec, ok := firstUnnamedExternalSpec(execution.TestCase.ExpectedResults); ok {
				rows, err := e.loadExternalRows(spec.ExternalFile)
				if err != nil {
					return nil, fmt.Errorf("failed to load expected results from external file: %w", err)
				}
				if err := e.validateVerifyResults(verifyResult, rows); err != nil {
					return nil, fmt.Errorf("verify query validation failed: %w", err)
				}
			}
		}

		// 5. Also apply table-level expected results strategies
		for _, spec := range execution.TestCase.ExpectedResults {
			if spec.TableName != "" { // only table-qualified specs
				if err := e.validateTableStateBySpec(execution.Transaction, spec); err != nil {
					return nil, fmt.Errorf("table state validation failed: %w", err)
				}
			}
		}

		return verifyResult, nil
	}

	// 4. Validate (暫定: 旧式 ExpectedResult を直接比較) または 外部ファイル参照の無名期待
	if result.QueryType == SelectQuery || hasReturningClause(execution.SQL) {
		if len(execution.TestCase.ExpectedResult) > 0 {
			if err := compareRowsSlice(execution.TestCase.ExpectedResult, result.Data); err != nil {
				return nil, fmt.Errorf("simple validation failed: %w", err)
			}
		} else if spec, ok := firstUnnamedExternalSpec(execution.TestCase.ExpectedResults); ok {
			rows, err := e.loadExternalRows(spec.ExternalFile)
			if err != nil {
				return nil, fmt.Errorf("failed to load expected results from external file: %w", err)
			}
			if err := compareRowsSlice(rows, result.Data); err != nil {
				return nil, fmt.Errorf("simple validation failed: %w", err)
			}
		}
	}

	// 5. Table-level ExpectedResults with strategies (pk-*, all) validation
	for _, spec := range execution.TestCase.ExpectedResults {
		if spec.TableName != "" { // only table-qualified specs
			if err := e.validateTableStateBySpec(execution.Transaction, spec); err != nil {
				return nil, fmt.Errorf("table state validation failed: %w", err)
			}
		}
	}

	return result, nil
}

// firstUnnamedExternalSpec finds an ExpectedResultSpec with empty TableName and non-empty ExternalFile
func firstUnnamedExternalSpec(specs []markdownparser.ExpectedResultSpec) (markdownparser.ExpectedResultSpec, bool) {
	for _, s := range specs {
		if s.TableName == "" && s.ExternalFile != "" {
			return s, true
		}
	}
	return markdownparser.ExpectedResultSpec{}, false
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
	// Guard against indefinite blocking by bounding query duration
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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
		// Load external rows for fixture if needed

		if fixture.ExternalFile != "" && len(fixture.Data) == 0 {
			rows, err := e.loadExternalRows(fixture.ExternalFile)
			if err != nil {
				return fmt.Errorf("failed to load fixture external file for table %s: %w", fixture.TableName, err)
			}
			fixture.Data = rows
		}

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
	case markdownparser.Upsert:
		return e.executeUpsert(tx, fixture)
	case markdownparser.Delete:
		return e.executeDelete(tx, fixture)
	default:
		return fmt.Errorf("%w: %s", snapsql.ErrUnsupportedInsertStrategy, fixture.Strategy)
	}
}

// loadExternalRows loads rows from an external YAML/JSON file path (relative to baseDir if not absolute)
func (e *Executor) loadExternalRows(path string) ([]map[string]any, error) {
	if path == "" {
		return nil, nil
	}
	p := path
	if !isAbsPath(p) && e.baseDir != "" {
		p = joinPath(e.baseDir, p)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	return unmarshalRows(b)
}

// Helpers for path and unmarshal
func isAbsPath(p string) bool        { return filepath.IsAbs(p) }
func joinPath(base, p string) string { return filepath.Clean(filepath.Join(base, p)) }

func unmarshalRows(b []byte) ([]map[string]any, error) {
	var rows []map[string]any
	if err := yaml.Unmarshal(b, &rows); err != nil {
		return nil, fmt.Errorf("failed to unmarshal external rows: %w", err)
	}
	// normalize values similar to markdownparser.normalizeValue
	for i := range rows {
		rows[i] = normalizeLoadedMap(rows[i])
	}
	return rows, nil
}

func normalizeLoadedMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = normalizeLoadedValue(v)
	}
	return out
}

func normalizeLoadedValue(v any) any {
	switch val := v.(type) {
	case float64:
		if float64(int64(val)) == val {
			return int64(val)
		}
		return val
	case float32:
		if float32(int32(val)) == val {
			return int32(val)
		}
		return val
	case []any:
		res := make([]any, len(val))
		for i, it := range val {
			res[i] = normalizeLoadedValue(it)
		}
		return res
	case map[any]any:
		res := make(map[string]any)
		for k, vv := range val {
			if ks, ok := k.(string); ok {
				res[ks] = normalizeLoadedValue(vv)
			}
		}
		return res
	case map[string]any:
		return normalizeLoadedMap(val)
	default:
		return v
	}
}

// executeClearInsert truncates the table and inserts data
func (e *Executor) executeClearInsert(tx *sql.Tx, fixture markdownparser.TableFixture) error {
	// 簡易DELETE実装（dialect依存truncateは未実装暫定）
	query := "DELETE FROM " + e.quoteIdentifier(fixture.TableName)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := tx.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to clear table %s: %w", fixture.TableName, err)
	}
	return e.insertData(tx, fixture.TableName, fixture.Data)
}

// executeInsert just inserts data into the table

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
		return fmt.Errorf("%w: %s", errTableInfoNotFound, fixture.TableName)
	}
	var pkCols []string
	for colName, colInfo := range tblInfo.Columns {
		if colInfo.IsPrimaryKey {
			pkCols = append(pkCols, colName)
		}
	}
	if len(pkCols) == 0 {
		return fmt.Errorf("%w: %s", errNoPrimaryKeyDefined, fixture.TableName)
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
				return fmt.Errorf("%w: %s (table %s)", errPrimaryKeyColumnMiss, pk, fixture.TableName)
			}
			whereClauses = append(whereClauses, fmt.Sprintf("%s = %s", e.quoteIdentifier(pk), e.getPlaceholder(idx)))
			values = append(values, val)
			idx++
		}
		// 主キー以外のカラムは無視

		query := fmt.Sprintf("DELETE FROM %s WHERE %s", e.quoteIdentifier(fixture.TableName), strings.Join(whereClauses, " AND "))
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
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
		return nil, fmt.Errorf("%w: %s", errTableInfoNotFound, tableName)
	}
	var pkCols []string
	for col, info := range tblInfo.Columns {
		if info.IsPrimaryKey {
			pkCols = append(pkCols, col)
		}
	}
	if len(pkCols) == 0 {
		return nil, fmt.Errorf("%w: %s", errNoPrimaryKeyDefined, tableName)
	}
	return pkCols, nil
}

// matchPrimaryKey: 主キー列で2行が一致するか
// (legacy helper matchPrimaryKey / extractPrimaryKey removed as unused)

// compareRowsSlice: 2つの[]map[string]anyを順序・件数・値で完全一致比較

func compareRowsSlice(expected, actual []map[string]any) error {
	if len(expected) != len(actual) {
		return fmt.Errorf("%w: expected %d got %d", errRowCountMismatch, len(expected), len(actual))
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
			return fmt.Errorf("%w: %s", errColumnMissing, k)
		}
		// 値比較特殊指定
		switch val := vExp.(type) {
		case []any:
			// [null], [notnull], [any], [regexp, ...]
			if len(val) == 1 {
				v0 := val[0]
				// YAML's bare 'null' becomes nil; treat as ["null"] matcher
				if v0 == nil {
					if vAct != nil {
						return fmt.Errorf("%w: column=%s got=%v", errExpectedNull, k, vAct)
					}
					break
				}
				switch v0 {
				case "null":
					if vAct != nil {
						return fmt.Errorf("%w: column=%s got=%v", errExpectedNull, k, vAct)
					}
				case "notnull":
					if vAct == nil {
						return fmt.Errorf("%w: column=%s", errExpectedNotNull, k)
					}
				case "any":
					// 何でもOK
				default:
					return fmt.Errorf("%w: column=%s matcher=%v", errUnknownMatcher, k, v0)
				}
			} else if len(val) == 2 && val[0] == "regexp" {
				pat, ok := val[1].(string)
				if !ok {
					return fmt.Errorf("%w: column=%s", errRegexpPatternType, k)
				}
				s, ok := vAct.(string)
				if !ok {
					return fmt.Errorf("%w: column=%s gotType=%T", errRegexpExpectString, k, vAct)
				}
				matched, err := regexp.MatchString(pat, s)
				if err != nil {
					return fmt.Errorf("column %s: regexp error: %w", k, err)
				}
				if !matched {
					return fmt.Errorf("%w: column=%s value=%s pattern=%s", errRegexpNotMatch, k, s, pat)
				}
			} else {
				return fmt.Errorf("%w: column=%s raw=%v", errInvalidMatcherSyntax, k, val)
			}
		default:
			// 通常値比較
			if !valueEquals(vExp, vAct) {
				return fmt.Errorf("%w: column=%s expected=%v got=%v", errValueMismatch, k, vExp, vAct)
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

	// Determine column list: if schema present use ColumnOrder intersection with row keys for determinism
	var columns []string
	tbl, hasSchema := e.tableInfo[tableName]

	if hasSchema && tbl != nil && len(tbl.ColumnOrder) > 0 {
		// validate & build list using first row's keys presence
		first := data[0]
		for _, c := range tbl.ColumnOrder {
			if _, ok := first[c]; ok { // only include present columns
				columns = append(columns, c)
			}
		}
		// verify required non-null columns exist in row
		for colName, colInfo := range tbl.Columns {
			if !colInfo.Nullable && !colInfo.IsPrimaryKey { // PK auto value許容
				if _, ok := first[colName]; !ok {
					return fmt.Errorf("%w: %s", errMissingRequiredColumn, colName)
				}
			}
		}
		// unknown column detection
		for k := range first {
			if _, ok := tbl.Columns[k]; !ok {
				return fmt.Errorf("%w: %s", errUnknownFixtureColumn, k)
			}
		}
	} else {
		for col := range data[0] {
			columns = append(columns, col)
		}
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	// Insert each row
	for _, row := range data {
		if hasSchema && tbl != nil {
			// per-row validation
			for colName, colInfo := range tbl.Columns {
				if !colInfo.Nullable && !colInfo.IsPrimaryKey {
					if _, ok := row[colName]; !ok {
						return fmt.Errorf("%w: %s", errMissingRequiredColumn, colName)
					}
				}
			}
			for k := range row {
				if _, ok := tbl.Columns[k]; !ok {
					return fmt.Errorf("%w: %s", errUnknownFixtureColumn, k)
				}
			}
		}
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
	pkCols, err := e.getPrimaryKeyColumns(fixture.TableName)
	if err != nil {
		return err
	}
	ctx := context.Background()
	for _, row := range fixture.Data {
		// スキーマ列順序使用。なければ行のキー集合
		var cols []string
		if tbl, ok := e.tableInfo[fixture.TableName]; ok && tbl != nil && len(tbl.ColumnOrder) > 0 {
			for _, c := range tbl.ColumnOrder {
				if _, exists := row[c]; exists {
					cols = append(cols, c)
				}
			}
			// validation (required columns & unknown columns)
			for name, info := range tbl.Columns {
				if !info.Nullable && !info.IsPrimaryKey {
					if _, ok := row[name]; !ok {
						return fmt.Errorf("%w: %s", errMissingRequiredColumn, name)
					}
				}
			}
			for k := range row {
				if _, ok := tbl.Columns[k]; !ok {
					return fmt.Errorf("%w: %s", errUnknownFixtureColumn, k)
				}
			}
		} else {
			for c := range row {
				cols = append(cols, c)
			}
		}
		var placeholders []string
		var values []any
		for i, c := range cols {
			placeholders = append(placeholders, e.getPlaceholder(i+1))
			values = append(values, row[c]) // 存在しなければ nil
		}
		// 全PK存在チェック
		for _, pk := range pkCols {
			// カラムリスト内にPKが無い場合はエラー
			found := false
			for _, c := range cols {
				if c == pk {
					found = true
					break
				}
			}
			if !found || row[pk] == nil {
				return fmt.Errorf("%w: %s", errUpsertMissingPK, pk)
			}
		}
		// ORDER deterministic (optional) – 現在 map iteration 順は未保証だがテスト目的では影響軽微
		// SET 句（PK以外）
		var setClauses []string
		for _, col := range cols {
			isPK := false
			for _, pk := range pkCols {
				if pk == col {
					isPK = true
					break
				}
			}
			if isPK {
				continue
			}
			setClauses = append(setClauses, fmt.Sprintf("%s = EXCLUDED.%s", e.quoteIdentifier(col), e.quoteIdentifier(col)))
		}
		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s) DO UPDATE SET %s",
			e.quoteIdentifier(fixture.TableName),
			joinQuoted(e, cols, ","),
			strings.Join(placeholders, ","),
			joinQuoted(e, pkCols, ","),
			strings.Join(setClauses, ", "))
		// execution without verbose debug logging
		if _, err := tx.ExecContext(ctx, query, values...); err != nil {
			return fmt.Errorf("postgres upsert failed: %w", err)
		}
	}
	return nil
}

// executeMySQLUpsert implements upsert for MySQL
func (e *Executor) executeMySQLUpsert(tx *sql.Tx, fixture markdownparser.TableFixture) error {
	pkCols, err := e.getPrimaryKeyColumns(fixture.TableName)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for _, row := range fixture.Data {
		var cols []string
		var placeholders []string
		var values []any
		for col, val := range row {
			cols = append(cols, col)
			placeholders = append(placeholders, "?")
			values = append(values, val)
		}
		if tbl, ok := e.tableInfo[fixture.TableName]; ok && tbl != nil {
			for name, info := range tbl.Columns {
				if !info.Nullable && !info.IsPrimaryKey {
					if _, ok := row[name]; !ok {
						return fmt.Errorf("%w: %s", errMissingRequiredColumn, name)
					}
				}
			}
			for k := range row {
				if _, ok := tbl.Columns[k]; !ok {
					return fmt.Errorf("%w: %s", errUnknownFixtureColumn, k)
				}
			}
		}
		if tbl, ok := e.tableInfo[fixture.TableName]; ok && tbl != nil && len(tbl.ColumnOrder) > 0 {
			cols, placeholders, values = reorderForSchema(tbl.ColumnOrder, cols, placeholders, values)
		}
		var setClauses []string
		for _, col := range cols {
			isPK := false
			for _, pk := range pkCols {
				if pk == col {
					isPK = true
					break
				}
			}
			if isPK {
				continue
			}
			setClauses = append(setClauses, fmt.Sprintf("%s=VALUES(%s)", e.quoteIdentifier(col), e.quoteIdentifier(col)))
		}
		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON DUPLICATE KEY UPDATE %s",
			e.quoteIdentifier(fixture.TableName),
			joinQuoted(e, cols, ","),
			strings.Join(placeholders, ","),
			strings.Join(setClauses, ", "))
		if _, err := tx.ExecContext(ctx, query, values...); err != nil {
			return fmt.Errorf("mysql upsert failed: %w", err)
		}
	}
	return nil
}

// executeSQLiteUpsert implements upsert for SQLite
func (e *Executor) executeSQLiteUpsert(tx *sql.Tx, fixture markdownparser.TableFixture) error {
	pkCols, err := e.getPrimaryKeyColumns(fixture.TableName)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for _, row := range fixture.Data {
		var cols []string
		var placeholders []string
		var values []any
		for col, val := range row {
			cols = append(cols, col)
			placeholders = append(placeholders, "?")
			values = append(values, val)
		}
		if tbl, ok := e.tableInfo[fixture.TableName]; ok && tbl != nil {
			for name, info := range tbl.Columns {
				if !info.Nullable && !info.IsPrimaryKey {
					if _, ok := row[name]; !ok {
						return fmt.Errorf("%w: %s", errMissingRequiredColumn, name)
					}
				}
			}
			for k := range row {
				if _, ok := tbl.Columns[k]; !ok {
					return fmt.Errorf("%w: %s", errUnknownFixtureColumn, k)
				}
			}
		}
		if tbl, ok := e.tableInfo[fixture.TableName]; ok && tbl != nil && len(tbl.ColumnOrder) > 0 {
			cols, placeholders, values = reorderForSchema(tbl.ColumnOrder, cols, placeholders, values)
		}
		var setClauses []string
		for _, col := range cols {
			isPK := false
			for _, pk := range pkCols {
				if pk == col {
					isPK = true
					break
				}
			}
			if isPK {
				continue
			}
			setClauses = append(setClauses, fmt.Sprintf("%s=excluded.%s", e.quoteIdentifier(col), e.quoteIdentifier(col)))
		}
		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON CONFLICT(%s) DO UPDATE SET %s",
			e.quoteIdentifier(fixture.TableName),
			joinQuoted(e, cols, ","),
			strings.Join(placeholders, ","),
			joinQuoted(e, pkCols, ","),
			strings.Join(setClauses, ", "))
		if _, err := tx.ExecContext(ctx, query, values...); err != nil {
			return fmt.Errorf("sqlite upsert failed: %w", err)
		}
	}
	return nil
}

// joinQuoted helper: quote and join identifiers
func joinQuoted(e *Executor, cols []string, sep string) string {
	var parts []string
	for _, c := range cols {
		parts = append(parts, e.quoteIdentifier(c))
	}
	return strings.Join(parts, sep)
}

// reorderForSchema: schema ColumnOrder に従い指定された並びに再構成（欠損カラムは末尾に残す）
func reorderForSchema(order []string, cols []string, placeholders []string, values []any) ([]string, []string, []any) {
	index := make(map[string]int, len(cols))
	for i, c := range cols {
		index[c] = i
	}
	var newCols []string
	var newPlace []string
	var newVals []any
	used := make(map[string]bool, len(cols))
	for _, o := range order {
		if idx, ok := index[o]; ok {
			newCols = append(newCols, cols[idx])
			newPlace = append(newPlace, placeholders[idx])
			newVals = append(newVals, values[idx])
			used[o] = true
		}
	}
	// append any remaining columns not in order
	for i, c := range cols {
		if !used[c] {
			newCols = append(newCols, c)
			newPlace = append(newPlace, placeholders[i])
			newVals = append(newVals, values[i])
		}
	}
	return newCols, newPlace, newVals
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
// validateDirectResults removed (unused)
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

// integrate table-level expected results validation into executeFullTest path
