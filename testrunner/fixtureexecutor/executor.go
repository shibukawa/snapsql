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
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/markdownparser"
	"github.com/shibukawa/snapsql/parser"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/query"
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
	keywordUpdateRegexp      = regexp.MustCompile(`\bUPDATE\b`)
	keywordDeleteRegexp      = regexp.MustCompile(`\bDELETE\b`)
	keywordInsertRegexp      = regexp.MustCompile(`\bINSERT\b`)
	errUnknownFixtureColumn  = errors.New("fixture row contains unknown column")
)

const maxTraceRows = 20

// SQLTrace captures executed statements for verbose output.
type SQLTrace struct {
	Label         string
	Statement     string
	Parameters    map[string]any
	QueryType     QueryType
	Rows          []map[string]any
	RowsTruncated bool
	TotalRows     int
	Args          []any
}

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
	Verbose  bool
}

// DefaultExecutionOptions returns default execution options
func DefaultExecutionOptions() *ExecutionOptions {
	return &ExecutionOptions{
		Mode:     FullTest,
		Commit:   false,
		Parallel: runtime.NumCPU(),
		Timeout:  2 * time.Minute,
		Verbose:  false,
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

func (qt QueryType) String() string {
	switch qt {
	case SelectQuery:
		return "SELECT"
	case InsertQuery:
		return "INSERT"
	case UpdateQuery:
		return "UPDATE"
	case DeleteQuery:
		return "DELETE"
	default:
		return "UNKNOWN"
	}
}

// ValidationResult contains the result of query execution and validation
type ValidationResult struct {
	Data         []map[string]any
	RowsAffected int64
	QueryType    QueryType
}

var (
	currentDateAnchorMu  sync.RWMutex
	currentDateAnchor    time.Time
	currentDateAnchorSet bool
)

func setCurrentDateAnchor(t time.Time) {
	currentDateAnchorMu.Lock()
	currentDateAnchor = t
	currentDateAnchorSet = true
	currentDateAnchorMu.Unlock()
}

func clearCurrentDateAnchor() {
	currentDateAnchorMu.Lock()
	currentDateAnchorSet = false
	currentDateAnchorMu.Unlock()
}

func currentDateAnchorNow() time.Time {
	currentDateAnchorMu.RLock()
	if currentDateAnchorSet {
		t := currentDateAnchor
		currentDateAnchorMu.RUnlock()
		return t
	}
	currentDateAnchorMu.RUnlock()
	return time.Now().UTC()
}

// ExpectedResultsStrategy defines comparison strategy for table state validation.
// Recognized values (design doc): "all" (default), "pk-match", "pk-exists", "pk-not-exists".
// Executor treats empty string as "all" for backward compatibility.

// TestExecution represents a single test execution context
type TestExecution struct {
	TestCase    *markdownparser.TestCase
	SQL         string         // SQL query from document
	Parameters  map[string]any // Parameters from test case
	Args        []any          // Positional arguments for PreparedSQL
	Options     *ExecutionOptions
	Transaction *sql.Tx
	Executor    *Executor
	Trace       []SQLTrace
	TimeAnchor  time.Time
}

func (te *TestExecution) addTrace(label, statement string, params map[string]any, args []any, result *ValidationResult) {
	if te == nil || te.Options == nil || !te.Options.Verbose {
		return
	}

	queryType := SelectQuery
	if result != nil {
		queryType = result.QueryType
	} else {
		queryType = detectQueryType(statement)
	}

	trace := SQLTrace{
		Label:      label,
		Statement:  statement,
		Parameters: copyParameters(params),
		QueryType:  queryType,
	}

	if len(args) > 0 {
		trace.Args = copyArgs(args)
	}

	if result != nil {
		trace.TotalRows = len(result.Data)
		if rows, truncated := sanitizeRows(result.Data); len(rows) > 0 {
			trace.Rows = rows
			trace.RowsTruncated = truncated
		} else if result.QueryType != SelectQuery && result.RowsAffected != 0 {
			trace.Rows = []map[string]any{{"rows_affected": result.RowsAffected}}
			trace.TotalRows = 1
		}
	}

	te.Trace = append(te.Trace, trace)
}

func copyParameters(params map[string]any) map[string]any {
	if len(params) == 0 {
		return nil
	}
	out := make(map[string]any, len(params))
	for k, v := range params {
		out[k] = v
	}
	return out
}

func copyArgs(args []any) []any {
	if len(args) == 0 {
		return nil
	}
	out := make([]any, len(args))
	copy(out, args)
	return out
}

// resolveExecutableSQL returns the statement actually executed against database/sql.
// The same string is propagated into SQL traces so verbose output mirrors the real execution.
func (e *Executor) resolveExecutableSQL(testCase *markdownparser.TestCase, rawSQL string) (string, []any) {
	if testCase == nil {
		return rawSQL, nil
	}

	if strings.TrimSpace(testCase.PreparedSQL) == "" {
		return rawSQL, nil
	}

	formatted := query.FormatSQLForDialect(testCase.PreparedSQL, e.dialect)
	args := copyArgs(testCase.SQLArgs)

	if strings.TrimSpace(formatted) == "" {
		return rawSQL, args
	}

	return formatted, args
}

func sanitizeRows(rows []map[string]any) ([]map[string]any, bool) {
	if len(rows) == 0 {
		return nil, false
	}
	truncated := false
	if len(rows) > maxTraceRows {
		rows = rows[:maxTraceRows]
		truncated = true
	}
	out := make([]map[string]any, len(rows))
	for i, row := range rows {
		copyRow := make(map[string]any, len(row))
		for k, v := range row {
			copyRow[k] = v
		}
		out[i] = copyRow
	}
	return out, truncated
}

func normalizeFixtureRows(rows []map[string]any) ([]map[string]any, error) {
	if len(rows) == 0 {
		return rows, nil
	}

	result := make([]map[string]any, len(rows))
	for i, row := range rows {
		conv, err := normalizeFixtureRow(row)
		if err != nil {
			return nil, err
		}
		result[i] = conv
	}

	return result, nil
}

func normalizeFixtureRow(row map[string]any) (map[string]any, error) {
	if row == nil {
		return nil, nil
	}

	result := make(map[string]any, len(row))
	for k, v := range row {
		nv, err := resolveFixtureValue(v)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve fixture value for %s: %w", k, err)
		}
		result[k] = nv
	}

	return result, nil
}

func resolveFixtureValue(value any) (any, error) {
	switch v := value.(type) {
	case []any:
		if len(v) == 0 {
			return value, nil
		}
		if len(v) == 1 {
			if v[0] == nil {
				return nil, nil
			}
			if str, ok := v[0].(string); ok {
				signature := strings.ToLower(strings.TrimSpace(str))
				if signature == "null" || signature == "nil" {
					return nil, nil
				}
			}
		}
		if first, ok := v[0].(string); ok {
			matcher := strings.ToLower(strings.TrimSpace(first))
			switch matcher {
			case "currentdate", "current_date":
				base := currentDateAnchorNow()
				offset := time.Duration(0)
				if len(v) >= 2 {
					if durStr, ok := v[1].(string); ok && strings.TrimSpace(durStr) != "" {
						d, err := parseFlexibleDuration(durStr)
						if err != nil {
							return nil, err
						}
						offset = d
					}
				}
				return base.Add(offset), nil
			}
		}

		resolved := make([]any, len(v))
		for i, elem := range v {
			val, err := resolveFixtureValue(elem)
			if err != nil {
				return nil, err
			}
			resolved[i] = val
		}
		return resolved, nil
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, elem := range v {
			val, err := resolveFixtureValue(elem)
			if err != nil {
				return nil, err
			}
			out[key] = val
		}
		return out, nil
	case string:
		if arr, ok := parseBracketLiteral(v); ok {
			return resolveFixtureValue(arr)
		}
		return v, nil
	default:
		return value, nil
	}
}

func parseBracketLiteral(raw string) ([]any, bool) {
	s := strings.TrimSpace(raw)
	if len(s) < 2 || s[0] != '[' || s[len(s)-1] != ']' {
		return nil, false
	}
	inner := strings.TrimSpace(s[1 : len(s)-1])
	if inner == "" {
		return []any{}, true
	}
	parts := strings.Split(inner, ",")
	result := make([]any, len(parts))
	for i, p := range parts {
		result[i] = strings.TrimSpace(p)
	}
	return result, true
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
func (e *Executor) ExecuteTest(testCase *markdownparser.TestCase, sql string, parameters map[string]any, opts *ExecutionOptions) (*ValidationResult, []SQLTrace, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, wrapDefinitionFailure(err, "failed to begin transaction")
	}

	defer func() {
		if opts.Commit {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	anchor := time.Now().UTC()
	setCurrentDateAnchor(anchor)
	defer clearCurrentDateAnchor()

	finalSQL, args := e.resolveExecutableSQL(testCase, sql)

	execution := &TestExecution{
		TestCase:    testCase,
		SQL:         finalSQL,
		Parameters:  parameters,
		Args:        args,
		Options:     opts,
		Transaction: tx,
		Executor:    e,
		TimeAnchor:  anchor,
	}

	if err := NormalizeParameters(execution.Parameters); err != nil {
		return nil, nil, wrapDefinitionFailure(err, "failed to normalize parameters")
	}

	result, err := e.executeTestSteps(execution)
	return result, execution.Trace, err
}

func formatArgsForContext(values []any) string {
	if len(values) == 0 {
		return ""
	}

	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = fmt.Sprintf("%v", v)
	}

	return strings.Join(parts, ",")
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
		if err := compareRowsSlice(spec.Data, actual, spec.TableName, pkCols, false, true); err != nil {
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
	result, err := e.executeQuery(execution, execution.SQL, execution.Parameters, execution.Args)
	if err != nil {
		return nil, wrapDefinitionFailure(err, "failed to execute query")
	}

	// Validate results if expected results are provided
	if len(execution.TestCase.ExpectedResult) > 0 {
		specs, err := parseValidationSpecs(execution.TestCase.ExpectedResult)
		if err != nil {
			return nil, wrapDefinitionFailure(err, "failed to parse validation specs")
		}

		if err := e.validateResult(execution.Transaction, result, specs); err != nil {
			return nil, wrapAssertionFailure(err, "validation failed")
		}
	}

	return result, nil
}

// executeFullTest executes the complete test flow
func (e *Executor) executeFullTest(execution *TestExecution) (*ValidationResult, error) {
	// 1. Execute fixtures
	if err := e.executeFixtures(execution.Transaction, execution.TestCase.Fixtures); err != nil {
		return nil, wrapDefinitionFailure(err, "failed to execute fixtures")
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
		if _, err := execution.Transaction.ExecContext(ctx, execution.SQL, execution.Args...); err != nil {
			// Fallback: run as QueryContext then close immediately without iteration
			if rows, qerr := execution.Transaction.QueryContext(ctx, execution.SQL, execution.Args...); qerr == nil {
				rows.Close()
			} else {
				execution.addTrace("main query", execution.SQL, execution.Parameters, execution.Args, nil)
				combined := fmt.Errorf("exec=%v query=%v", err, qerr)
				return nil, wrapDefinitionFailure(combined, "failed to execute main SQL (exec/query fallback)")
			}
		}
		result = &ValidationResult{Data: nil, RowsAffected: 0, QueryType: SelectQuery}
		execution.addTrace("main query", execution.SQL, execution.Parameters, execution.Args, result)
	} else {
		var err error
		result, err = e.executeQuery(execution, execution.SQL, execution.Parameters, execution.Args)
		if err != nil {
			return nil, wrapDefinitionFailure(err, "failed to execute query")
		}
	}

	// 3. Execute verify query if present
	if execution.TestCase.VerifyQuery != "" {
		verifyResult, err := e.executeVerifyQuery(execution, execution.TestCase.VerifyQuery)
		if err != nil {
			return nil, wrapDefinitionFailure(err, "failed to execute verify query")
		}

		// 4. Validate verify query results (legacy unnamed)
		if len(execution.TestCase.ExpectedResult) > 0 {
			if err := e.validateVerifyResults(verifyResult, execution.TestCase.ExpectedResult); err != nil {
				return nil, wrapAssertionFailure(err, "verify query validation failed")
			}
		} else {
			// Support unnamed external expected results via ExpectedResults entry with empty TableName
			if spec, ok := firstUnnamedExternalSpec(execution.TestCase.ExpectedResults); ok {
				rows, err := e.loadExternalRows(spec.ExternalFile)
				if err != nil {
					return nil, wrapDefinitionFailure(err, "failed to load expected results from external file")
				}
				if err := e.validateVerifyResults(verifyResult, rows); err != nil {
					return nil, wrapAssertionFailure(err, "verify query validation failed")
				}
			}
		}

		// 5. Also apply table-level expected results strategies
		for _, spec := range execution.TestCase.ExpectedResults {
			if spec.TableName != "" { // only table-qualified specs
				if err := e.validateTableStateBySpec(execution.Transaction, spec); err != nil {
					return nil, wrapAssertionFailure(err, "table state validation failed")
				}
			}
		}

		return verifyResult, nil
	}

	// 4. Validate (暫定: 旧式 ExpectedResult を直接比較) または 外部ファイル参照の無名期待
	if result.QueryType == SelectQuery || hasReturningClause(execution.SQL) {
		if len(execution.TestCase.ExpectedResult) > 0 {
			if err := compareRowsSlice(execution.TestCase.ExpectedResult, result.Data, "", nil, execution.TestCase.ResultOrdered, true); err != nil {
				return nil, wrapAssertionFailure(err, "simple validation failed")
			}
		} else if spec, ok := firstUnnamedExternalSpec(execution.TestCase.ExpectedResults); ok {
			rows, err := e.loadExternalRows(spec.ExternalFile)
			if err != nil {
				return nil, wrapDefinitionFailure(err, "failed to load expected results from external file")
			}
			if err := compareRowsSlice(rows, result.Data, "", nil, execution.TestCase.ResultOrdered, true); err != nil {
				return nil, wrapAssertionFailure(err, "simple validation failed")
			}
		}
	}

	// 5. Table-level ExpectedResults with strategies (pk-*, all) validation
	for _, spec := range execution.TestCase.ExpectedResults {
		if spec.TableName != "" { // only table-qualified specs
			if err := e.validateTableStateBySpec(execution.Transaction, spec); err != nil {
				return nil, wrapAssertionFailure(err, "table state validation failed")
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
	trimmed := strings.TrimSpace(sql)
	if trimmed == "" {
		return SelectQuery
	}

	if qt, ok := detectQueryTypeFromParser(trimmed); ok {
		return qt
	}

	upper := strings.ToUpper(trimmed)
	if strings.HasPrefix(upper, "SELECT") {
		return SelectQuery
	}
	if strings.HasPrefix(upper, "INSERT") {
		return InsertQuery
	}
	if strings.HasPrefix(upper, "UPDATE") {
		return UpdateQuery
	}
	if strings.HasPrefix(upper, "DELETE") {
		return DeleteQuery
	}

	if strings.HasPrefix(upper, "WITH") {
		if containsKeyword(upper, "UPDATE") {
			return UpdateQuery
		}
		if containsKeyword(upper, "DELETE") {
			return DeleteQuery
		}
		if containsKeyword(upper, "INSERT") {
			return InsertQuery
		}
		return SelectQuery
	}

	// Default to SELECT for unknown queries
	return SelectQuery
}

func detectQueryTypeFromParser(sql string) (QueryType, bool) {
	stmt, _, err := parser.ParseSQLFile(strings.NewReader(sql), nil, "detect_query.sql", "", parser.DefaultOptions)
	if err != nil || stmt == nil {
		return 0, false
	}

	switch stmt.Type() {
	case cmn.SELECT_STATEMENT:
		return SelectQuery, true
	case cmn.INSERT_INTO_STATEMENT:
		return InsertQuery, true
	case cmn.UPDATE_STATEMENT:
		return UpdateQuery, true
	case cmn.DELETE_FROM_STATEMENT:
		return DeleteQuery, true
	default:
		return 0, false
	}
}

func containsKeyword(sqlUpper, keyword string) bool {
	switch keyword {
	case "UPDATE":
		return keywordUpdateRegexp.MatchString(sqlUpper)
	case "DELETE":
		return keywordDeleteRegexp.MatchString(sqlUpper)
	case "INSERT":
		return keywordInsertRegexp.MatchString(sqlUpper)
	default:
		return false
	}
}

// hasReturningClause checks if the SQL query has a RETURNING clause
func hasReturningClause(sql string) bool {
	// Convert to uppercase and check for RETURNING keyword
	upperSQL := strings.ToUpper(sql)
	return strings.Contains(upperSQL, "RETURNING")
}

// executeQuery executes the SQL query and returns the result
func (e *Executor) executeQuery(execution *TestExecution, sqlQuery string, parameters map[string]any, args []any) (*ValidationResult, error) {
	queryType := detectQueryType(sqlQuery)
	trx := execution.Transaction

	// Parameter replacement in SQL query is handled by the template engine
	// For now, execute the query as-is

	// Check for RETURNING clause in DML queries
	if (queryType == InsertQuery || queryType == UpdateQuery || queryType == DeleteQuery) && hasReturningClause(sqlQuery) {
		// Execute as SELECT query to get returned data
		label := fmt.Sprintf("%s query with RETURNING", queryType.String())
		result, err := e.executeSelectQuery(trx, sqlQuery, args, label)
		if err != nil {
			execution.addTrace("main query", sqlQuery, parameters, args, nil)
			return nil, err
		}
		// Keep the original query type for validation logic
		result.QueryType = queryType
		execution.addTrace("main query", sqlQuery, parameters, args, result)

		return result, nil
	}

	switch queryType {
	case SelectQuery:
		result, err := e.executeSelectQuery(trx, sqlQuery, args, "SELECT query")
		if err != nil {
			execution.addTrace("main query", sqlQuery, parameters, args, nil)
			return nil, err
		}
		execution.addTrace("main query", sqlQuery, parameters, args, result)
		return result, nil
	case InsertQuery, UpdateQuery, DeleteQuery:
		result, err := e.executeDMLQuery(trx, sqlQuery, queryType, args)
		if err != nil {
			execution.addTrace("main query", sqlQuery, parameters, args, nil)
			return nil, err
		}
		execution.addTrace("main query", sqlQuery, parameters, args, result)
		return result, nil
	default:
		return nil, snapsql.ErrUnsupportedQueryType
	}
}

// executeSelectQuery executes a query expected to return rows and returns the data.
// label describes the originating statement (e.g., "SELECT query", "UPDATE query with RETURNING").
func (e *Executor) executeSelectQuery(tx *sql.Tx, sqlQuery string, args []any, label string) (*ValidationResult, error) {
	// Guard against indefinite blocking by bounding query duration
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if label == "" {
		label = "SELECT query"
	}

	rows, err := tx.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute %s: %w", label, err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get column names for %s: %w", label, err)
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
			return nil, fmt.Errorf("failed to scan row for %s: %w", label, err)
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
		return nil, fmt.Errorf("error iterating rows for %s: %w", label, err)
	}

	return &ValidationResult{
		Data:         data,
		RowsAffected: int64(len(data)),
		QueryType:    SelectQuery,
	}, nil
}

// executeDMLQuery executes INSERT/UPDATE/DELETE queries and returns affected rows
func (e *Executor) executeDMLQuery(tx *sql.Tx, sqlQuery string, queryType QueryType, args []any) (*ValidationResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := tx.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, wrapDefinitionFailure(err, "failed to execute DML query")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, wrapDefinitionFailure(err, "failed to get rows affected")
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
		ctx := map[string]string{"table": fixture.TableName}
		if fixture.Strategy != "" {
			ctx["strategy"] = string(fixture.Strategy)
		}

		if fixture.Line > 0 {
			ctx["line"] = strconv.Itoa(fixture.Line)
		}

		if fixture.ExternalFile != "" && len(fixture.Data) == 0 {
			rows, err := e.loadExternalRows(fixture.ExternalFile)
			if err != nil {
				return wrapDefinitionFailureWithContext(ctx, err, "failed to load fixture external file for table %s", fixture.TableName)
			}
			fixture.Data = rows
		}

		err := e.executeTableFixture(tx, fixture)
		if err != nil {
			return wrapDefinitionFailureWithContext(ctx, err, "failed to execute fixture for table %s", fixture.TableName)
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
		return wrapDefinitionFailureWithContext(map[string]string{"table": fixture.TableName, "operation": "clear"}, err, "failed to clear table %s", fixture.TableName)
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

// compareRowsSlice: 2つの[]map[string]anyを比較
// - orderSensitive: 並び順も検証
// - ignoreUnexpected: 実際の行に期待にない余分なカラムがあっても許容（シンプル検証用）

func compareRowsSlice(expected, actual []map[string]any, table string, pkCols []string, orderSensitive bool, ignoreUnexpected bool) error {
	if orderSensitive || len(pkCols) == 0 {
		return compareRowsSliceOrdered(expected, actual, table, pkCols, ignoreUnexpected)
	}

	if !rowsContainPrimaryColumns(expected, pkCols) || !rowsContainPrimaryColumns(actual, pkCols) {
		return compareRowsSliceOrdered(expected, actual, table, pkCols, ignoreUnexpected)
	}

	diff := &DiffError{Table: table, PrimaryKeys: pkCols}

	expectedIndex := indexRowsByPK(expected, pkCols)
	actualIndex := indexRowsByPK(actual, pkCols)

	for key, expRows := range expectedIndex {
		actRows := actualIndex[key]

		for i, expRow := range expRows {
			if i < len(actRows) {
				colDiffs := collectRowDiffs(expRow, actRows[i], ignoreUnexpected)
				if len(colDiffs) > 0 {
					keyLabel := buildRowKey(pkCols, expRow, actRows[i], i)
					diff.RowDiffs = append(diff.RowDiffs, RowDiff{Key: keyLabel, Diffs: colDiffs})
				}
			} else {
				keyLabel := buildRowKey(pkCols, expRow, nil, i)
				diff.RowDiffs = append(diff.RowDiffs, RowDiff{
					Key:       keyLabel,
					RowStatus: "missing",
					Diffs: []ColumnDiff{{
						Column:   "__row__",
						Expected: formatRowForDiff(expRow),
						Actual:   "<missing>",
					}},
				})
			}
		}

		if len(actRows) > len(expRows) {
			for i := len(expRows); i < len(actRows); i++ {
				keyLabel := buildRowKey(pkCols, nil, actRows[i], i)
				diff.RowDiffs = append(diff.RowDiffs, RowDiff{
					Key:       keyLabel,
					RowStatus: "unexpected",
					Diffs: []ColumnDiff{{
						Column:   "__row__",
						Expected: "<missing>",
						Actual:   formatRowForDiff(actRows[i]),
					}},
				})
			}
		}

		delete(actualIndex, key)
	}

	for _, remaining := range actualIndex {
		for i, row := range remaining {
			keyLabel := buildRowKey(pkCols, nil, row, i)
			diff.RowDiffs = append(diff.RowDiffs, RowDiff{
				Key:       keyLabel,
				RowStatus: "unexpected",
				Diffs: []ColumnDiff{{
					Column:   "__row__",
					Expected: "<missing>",
					Actual:   formatRowForDiff(row),
				}},
			})
		}
	}

	if len(expected) != len(actual) {
		diff.RowCountMismatch = true
		diff.ExpectedRows = len(expected)
		diff.ActualRows = len(actual)
	}

	if diff.RowCountMismatch || len(diff.RowDiffs) > 0 {
		return diff
	}

	return nil
}

func compareRowsSliceOrdered(expected, actual []map[string]any, table string, pkCols []string, ignoreUnexpected bool) error {
	diff := &DiffError{Table: table, PrimaryKeys: pkCols}
	minLen := len(expected)
	if len(actual) < minLen {
		minLen = len(actual)
	}

	for i := 0; i < minLen; i++ {
		colDiffs := collectRowDiffs(expected[i], actual[i], ignoreUnexpected)
		if len(colDiffs) > 0 {
			key := buildRowKey(pkCols, expected[i], actual[i], i)
			diff.RowDiffs = append(diff.RowDiffs, RowDiff{Key: key, Diffs: colDiffs})
		}
	}

	if len(expected) != len(actual) {
		diff.RowCountMismatch = true
		diff.ExpectedRows = len(expected)
		diff.ActualRows = len(actual)
	}

	if len(expected) > len(actual) {
		for i := len(actual); i < len(expected); i++ {
			key := buildRowKey(pkCols, expected[i], nil, i)
			diff.RowDiffs = append(diff.RowDiffs, RowDiff{
				Key:       key,
				RowStatus: "missing",
				Diffs: []ColumnDiff{{
					Column:   "__row__",
					Expected: formatRowForDiff(expected[i]),
					Actual:   "<missing>",
				}},
			})
		}
	} else if len(actual) > len(expected) {
		for i := len(expected); i < len(actual); i++ {
			key := buildRowKey(pkCols, nil, actual[i], i)
			diff.RowDiffs = append(diff.RowDiffs, RowDiff{
				Key:       key,
				RowStatus: "unexpected",
				Diffs: []ColumnDiff{{
					Column:   "__row__",
					Expected: "<missing>",
					Actual:   formatRowForDiff(actual[i]),
				}},
			})
		}
	}

	if diff.RowCountMismatch || len(diff.RowDiffs) > 0 {
		return diff
	}

	return nil
}

func collectRowDiffs(expected, actual map[string]any, ignoreUnexpected bool) []ColumnDiff {
	diffs := make([]ColumnDiff, 0)
	seen := make(map[string]struct{})
	for k, vExp := range expected {
		seen[k] = struct{}{}
		vAct, ok := actual[k]
		if !ok {
			diffs = append(diffs, ColumnDiff{Column: k, Expected: formatValueForDiff(vExp), Actual: "<missing>", Reason: "missing column"})
			continue
		}
		if matchDiff := evaluateMatcherDiff(k, vExp, vAct); matchDiff != nil {
			diffs = append(diffs, *matchDiff)
		}
	}
	for k, vAct := range actual {
		if _, ok := seen[k]; ok {
			continue
		}
		if !ignoreUnexpected {
			diffs = append(diffs, ColumnDiff{Column: k, Expected: "<not provided>", Actual: formatValueForDiff(vAct), Reason: "unexpected column"})
		}
	}
	return diffs
}

func rowsContainPrimaryColumns(rows []map[string]any, pkCols []string) bool {
	if len(pkCols) == 0 {
		return false
	}
	for _, row := range rows {
		for _, col := range pkCols {
			if _, ok := row[col]; !ok {
				return false
			}
		}
	}
	return true
}

func indexRowsByPK(rows []map[string]any, pkCols []string) map[string][]map[string]any {
	indexed := make(map[string][]map[string]any, len(rows))
	for _, row := range rows {
		key := buildPrimaryKeyString(pkCols, row)
		indexed[key] = append(indexed[key], row)
	}
	return indexed
}

func buildPrimaryKeyString(pkCols []string, row map[string]any) string {
	parts := make([]string, len(pkCols))
	for i, col := range pkCols {
		parts[i] = fmt.Sprintf("%s=%v", col, formatValueForDiff(row[col]))
	}
	return strings.Join(parts, ",")
}

func evaluateMatcherDiff(column string, expected any, actual any) *ColumnDiff {
	switch val := expected.(type) {
	case []any:
		if len(val) >= 1 {
			switch first := val[0].(type) {
			case nil:
				if actual != nil {
					return &ColumnDiff{Column: column, Expected: "[null]", Actual: formatValueForDiff(actual), Reason: "expected null"}
				}
				return nil
			case string:
				matcher := strings.ToLower(first)
				switch matcher {
				case "currentdate", "current_date":
					expectedTime, tolerance, display, err := evaluateRelativeTimeMatcher(val)
					if err != nil {
						return &ColumnDiff{Column: column, Expected: formatValueForDiff(val), Actual: formatValueForDiff(actual), Reason: err.Error()}
					}

					actualTime, ok := parseTimeValue(actual)
					if !ok {
						return &ColumnDiff{Column: column, Expected: display, Actual: formatValueForDiff(actual), Reason: "invalid time value"}
					}

					delta := actualTime.Sub(expectedTime)
					if delta < 0 {
						delta = -delta
					}

					if delta > tolerance {
						return &ColumnDiff{Column: column, Expected: display, Actual: actualTime.UTC().Format(time.RFC3339), Reason: "timestamp outside tolerance"}
					}
					return nil
				case "null":
					if actual != nil {
						return &ColumnDiff{Column: column, Expected: "[null]", Actual: formatValueForDiff(actual), Reason: "expected null"}
					}
					return nil
				case "notnull":
					if actual == nil {
						return &ColumnDiff{Column: column, Expected: "[notnull]", Actual: "<null>", Reason: "expected value"}
					}
					return nil
				case "any":
					return nil
				case "regexp":
					if len(val) == 2 {
						pat, _ := val[1].(string)
						s, ok := actual.(string)
						if !ok {
							return &ColumnDiff{Column: column, Expected: fmt.Sprintf("[regexp,%s]", pat), Actual: formatValueForDiff(actual), Reason: "expected string"}
						}
						matched, err := regexp.MatchString(pat, s)
						if err != nil || !matched {
							return &ColumnDiff{Column: column, Expected: fmt.Sprintf("[regexp,%s]", pat), Actual: s, Reason: "regex mismatch"}
						}
						return nil
					}
				}
			default:
				return &ColumnDiff{Column: column, Expected: formatValueForDiff(expected), Actual: formatValueForDiff(actual), Reason: "invalid matcher"}
			}
		}
		return &ColumnDiff{Column: column, Expected: formatValueForDiff(expected), Actual: formatValueForDiff(actual), Reason: "invalid matcher"}
	default:
		if !valueEquals(expected, actual) {
			return &ColumnDiff{Column: column, Expected: formatValueForDiff(expected), Actual: formatValueForDiff(actual), Reason: "value mismatch"}
		}
	}
	return nil
}

func evaluateRelativeTimeMatcher(arr []any) (time.Time, time.Duration, string, error) {
	base := currentDateAnchorNow()
	offset := time.Duration(0)
	tolerance := time.Minute
	offsetToken := ""
	toleranceToken := ""

	if len(arr) >= 2 {
		if token, ok := arr[1].(string); ok && strings.TrimSpace(token) != "" {
			trimmed := strings.TrimSpace(token)
			dur, err := parseFlexibleDuration(trimmed)
			if err != nil {
				return time.Time{}, 0, "", err
			}
			offset = dur
			offsetToken = trimmed
		}
	}

	if len(arr) >= 3 {
		if token, ok := arr[2].(string); ok && strings.TrimSpace(token) != "" {
			trimmed := strings.TrimSpace(token)
			dur, err := parseFlexibleDuration(trimmed)
			if err != nil {
				return time.Time{}, 0, "", err
			}
			if dur < 0 {
				dur = -dur
			}
			tolerance = dur
			toleranceToken = trimmed
		}
	}

	display := "[currentdate]"
	if offsetToken != "" && toleranceToken != "" {
		display = fmt.Sprintf("[currentdate,%s,%s]", offsetToken, toleranceToken)
	} else if offsetToken != "" {
		display = fmt.Sprintf("[currentdate,%s]", offsetToken)
	} else if toleranceToken != "" {
		display = fmt.Sprintf("[currentdate,%s]", toleranceToken)
	}

	return base.Add(offset), tolerance, display, nil
}

func buildRowKey(pkCols []string, expected, actual map[string]any, index int) map[string]any {
	key := make(map[string]any)
	if len(pkCols) == 0 {
		key["row_index"] = index
		return key
	}
	source := expected
	if source == nil {
		source = actual
	}
	for _, col := range pkCols {
		if source != nil {
			key[col] = formatValueForDiff(source[col])
		} else {
			key[col] = "<nil>"
		}
	}
	return key
}

func formatRowForDiff(row map[string]any) string {
	if row == nil {
		return "<nil>"
	}
	parts := make([]string, 0, len(row))
	for k, v := range row {
		parts = append(parts, fmt.Sprintf("%s=%v", k, formatValueForDiff(v)))
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

func formatValueForDiff(v any) any {
	switch val := v.(type) {
	case []byte:
		return string(val)
	default:
		return val
	}
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
			if len(val) >= 1 {
				switch first := val[0].(type) {
				case nil:
					if vAct != nil {
						return fmt.Errorf("%w: column=%s got=%v", errExpectedNull, k, vAct)
					}
				case string:
					matcher := strings.ToLower(first)
					switch matcher {
					case "currentdate", "current_date":
						tolerance := time.Minute
						if len(val) >= 2 {
							if durStr, ok := val[1].(string); ok && durStr != "" {
								if parsed, err := time.ParseDuration(durStr); err == nil {
									tolerance = parsed
								}
							}
						}

						actualTime, ok := parseTimeValue(vAct)
						if !ok {
							return fmt.Errorf("%w: column=%s value=%v", errInvalidMatcherSyntax, k, vAct)
						}
						if durationAbs(time.Since(actualTime)) > tolerance {
							return fmt.Errorf("%w: column=%s value=%s tolerance=%s", errValueMismatch, k, actualTime.UTC().Format(time.RFC3339), tolerance.String())
						}
						break
					case "null":
						if vAct != nil {
							return fmt.Errorf("%w: column=%s got=%v", errExpectedNull, k, vAct)
						}
					case "notnull":
						if vAct == nil {
							return fmt.Errorf("%w: column=%s", errExpectedNotNull, k)
						}
					case "any":
					// anything ok
					case "regexp":
						if len(val) != 2 {
							return fmt.Errorf("%w: column=%s raw=%v", errInvalidMatcherSyntax, k, val)
						}
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
					default:
						return fmt.Errorf("%w: column=%s matcher=%v", errUnknownMatcher, k, matcher)
					}
				default:
					return fmt.Errorf("%w: column=%s raw=%v", errInvalidMatcherSyntax, k, val)
				}
				break
			}
			return fmt.Errorf("%w: column=%s raw=%v", errInvalidMatcherSyntax, k, val)
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
			if fa == fb {
				return true
			}
			if math.Abs(fa-fb) < 1e-9 {
				return true
			}
			return false
		}
	}

	if ta, ok := parseTimeValue(a); ok {
		if tb, ok2 := parseTimeValue(b); ok2 {
			return ta.Equal(tb)
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

var timeLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02 15:04:05Z07:00",
	"2006-01-02 15:04:05 -0700 MST",
	"2006-01-02 15:04:05 -0700",
	"2006-01-02 15:04:05",
}

func parseTimeValue(v any) (time.Time, bool) {
	switch val := v.(type) {
	case time.Time:
		return val, true
	case string:
		return parseTimeString(val)
	case []byte:
		return parseTimeString(string(val))
	default:
		return time.Time{}, false
	}
}

func parseTimeString(raw string) (time.Time, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return time.Time{}, false
	}
	for _, layout := range timeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	if strings.Contains(s, " ") && !strings.Contains(s, "T") {
		replaced := strings.Replace(s, " ", "T", 1)
		if t, err := time.Parse(time.RFC3339Nano, replaced); err == nil {
			return t, true
		}
		if t, err := time.Parse(time.RFC3339, replaced); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func durationAbs(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

func parseFlexibleDuration(raw string) (time.Duration, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, fmt.Errorf("duration must start with + or -: %s", raw)
	}
	sign := 1
	switch s[0] {
	case '+':
		s = s[1:]
	case '-':
		sign = -1
		s = s[1:]
	default:
		return 0, fmt.Errorf("duration must start with + or -: %s", raw)
	}

	if len(strings.TrimSpace(s)) == 0 {
		return 0, fmt.Errorf("invalid duration: %s", raw)
	}

	if strings.HasSuffix(s, "d") {
		val := strings.TrimSuffix(s, "d")
		f, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid day duration: %s", raw)
		}
		d := time.Duration(f * 24 * float64(time.Hour))
		return time.Duration(sign) * d, nil
	}

	if dur, err := time.ParseDuration(s); err == nil {
		return time.Duration(sign) * dur, nil
	}

	return 0, fmt.Errorf("invalid duration: %s", raw)
}

// insertData inserts data into a table
func (e *Executor) insertData(tx *sql.Tx, tableName string, data []map[string]any) error {
	if len(data) == 0 {
		return nil
	}

	data, err := normalizeFixtureRows(data)
	if err != nil {
		return wrapDefinitionFailureWithContext(map[string]string{"table": tableName, "operation": "normalize"}, err, "failed to normalize fixture row")
	}

	// Determine column list: if schema present use ColumnOrder intersection with row keys for determinism
	var columns []string
	tbl, hasSchema := e.tableInfo[tableName]

	// Collect union of columns present across all rows
	columnSet := make(map[string]struct{})
	for _, row := range data {
		for col := range row {
			columnSet[col] = struct{}{}
		}
	}

	if hasSchema && tbl != nil && len(tbl.ColumnOrder) > 0 {
		// unknown column detection against schema
		for col := range columnSet {
			if _, ok := tbl.Columns[col]; !ok {
				return fmt.Errorf("%w: %s", errUnknownFixtureColumn, col)
			}
		}

		for _, c := range tbl.ColumnOrder {
			if _, ok := columnSet[c]; ok {
				columns = append(columns, c)
			}
		}
	} else {
		columns = make([]string, 0, len(columnSet))
		for col := range columnSet {
			columns = append(columns, col)
		}
		sort.Strings(columns)
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
		ctxMap := map[string]string{
			"table":     tableName,
			"operation": "prepare",
			"sql":       query,
		}
		ctxMap["columns"] = strings.Join(columns, ",")
		return wrapDefinitionFailureWithContext(ctxMap, err, "failed to prepare insert statement")
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
			ctxMap := map[string]string{
				"table":     tableName,
				"operation": "insert",
				"sql":       query,
				"args":      formatArgsForContext(values),
			}
			return wrapDefinitionFailureWithContext(ctxMap, err, "failed to insert row")
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
	rows, err := normalizeFixtureRows(fixture.Data)
	if err != nil {
		return fmt.Errorf("postgres upsert failed: %w", err)
	}
	for _, row := range rows {
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
	rows, err := normalizeFixtureRows(fixture.Data)
	if err != nil {
		return fmt.Errorf("mysql upsert failed: %w", err)
	}
	for _, row := range rows {
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
	rows, err := normalizeFixtureRows(fixture.Data)
	if err != nil {
		return fmt.Errorf("sqlite upsert failed: %w", err)
	}
	for _, row := range rows {
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
	case "postgres", "postgresql", "pg", "pgx":
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
	case "postgres", "postgresql", "pg", "pgx":
		return fmt.Sprintf("$%d", position)
	case "mysql", "sqlite":
		return "?"
	default:
		return "?"
	}
}

// executeVerifyQuery executes the verify query and returns the result
func (e *Executor) executeVerifyQuery(execution *TestExecution, verifyQuery string) (*ValidationResult, error) {
	// Split multiple queries by semicolon
	queries := e.parseMultipleQueries(verifyQuery)

	var allResults []map[string]any

	for _, query := range queries {
		if strings.TrimSpace(query) == "" {
			continue
		}

		result, err := e.executeSelectQuery(execution.Transaction, query, nil, "verify query")
		if err != nil {
			execution.addTrace("verify query", query, nil, nil, nil)
			return nil, fmt.Errorf("failed to execute verify query: %w", err)
		}

		allResults = append(allResults, result.Data...)
		execution.addTrace("verify query", query, nil, nil, result)
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
