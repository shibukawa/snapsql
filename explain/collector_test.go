package explain

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestCollectPostgresJSON(t *testing.T) {
	db := newStubDB(t, "EXPLAIN (ANALYZE true, VERBOSE true, FORMAT JSON) SELECT 1", []string{"QUERY PLAN"}, [][]driver.Value{
		{`[{"Plan":{"Node Type":"Seq Scan"}}]`},
	})
	defer db.Close()

	doc, err := Collect(t.Context(), CollectorOptions{
		DB:      db,
		Dialect: "postgres",
		SQL:     "SELECT 1",
		Analyze: true,
	})
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	if len(doc.RawJSON) == 0 {
		t.Fatalf("expected RawJSON to be populated")
	}
}

func TestCollectMySQLJSON(t *testing.T) {
	db := newStubDB(t, "EXPLAIN FORMAT=JSON SELECT 1", []string{"EXPLAIN"}, [][]driver.Value{
		{`{"query_block":{}}`},
	})
	defer db.Close()

	doc, err := Collect(t.Context(), CollectorOptions{
		DB:      db,
		Dialect: "mysql",
		SQL:     "SELECT 1",
	})
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	if len(doc.RawJSON) == 0 {
		t.Fatalf("expected RawJSON to be populated")
	}
}

func TestCollectSQLitePlan(t *testing.T) {
	db := newStubDB(t, "EXPLAIN QUERY PLAN SELECT 1", []string{"selectid", "order", "from", "detail"}, [][]driver.Value{
		{int64(0), int64(0), int64(0), "SCAN TABLE users"},
		{int64(0), int64(1), int64(0), "USE TEMP B-TREE"},
	})
	defer db.Close()

	doc, err := Collect(t.Context(), CollectorOptions{
		DB:      db,
		Dialect: "sqlite",
		SQL:     "SELECT 1",
	})
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	if doc.RawText == "" {
		t.Fatalf("expected RawText to be populated")
	}

	if !strings.Contains(doc.RawText, "SCAN TABLE users") {
		t.Fatalf("unexpected RawText: %s", doc.RawText)
	}
}

func TestCollectUnsupportedDialect(t *testing.T) {
	db := newStubDB(t, "", nil, nil)
	defer db.Close()

	_, err := Collect(t.Context(), CollectorOptions{
		DB:      db,
		Dialect: "oracle",
		SQL:     "SELECT 1",
	})
	if !errors.Is(err, ErrUnsupportedDialect) {
		t.Fatalf("expected ErrUnsupportedDialect, got %v", err)
	}
}

func TestCollectMissingDB(t *testing.T) {
	_, err := Collect(t.Context(), CollectorOptions{Dialect: "postgres"})
	if !errors.Is(err, ErrNoDatabase) {
		t.Fatalf("expected ErrNoDatabase, got %v", err)
	}
}

func TestCollectWithTimeout(t *testing.T) {
	db := newStubDB(t, "EXPLAIN FORMAT=JSON SELECT 1", []string{"EXPLAIN"}, [][]driver.Value{
		{`{"query_block":{}}`},
	})
	defer db.Close()

	doc, err := Collect(t.Context(), CollectorOptions{
		DB:      db,
		Dialect: "mysql",
		SQL:     "SELECT 1",
		Timeout: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	if len(doc.RawJSON) == 0 {
		t.Fatalf("expected RawJSON to be populated")
	}
}

// stub implementations -------------------------------------------------------

type stubConnector struct {
	t             *testing.T
	expectedQuery string
	columns       []string
	rows          [][]driver.Value
}

func (c *stubConnector) Connect(ctx context.Context) (driver.Conn, error) {
	return &stubConn{connector: c}, nil
}

func (c *stubConnector) Driver() driver.Driver {
	return stubDriver{}
}

type stubDriver struct{}

func (stubDriver) Open(name string) (driver.Conn, error) {
	return nil, errors.New("should not be called")
}

type stubConn struct {
	connector *stubConnector
}

var _ driver.QueryerContext = (*stubConn)(nil)
var _ driver.Conn = (*stubConn)(nil)

func (c *stubConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not implemented") }
func (c *stubConn) Close() error                        { return nil }
func (c *stubConn) Begin() (driver.Tx, error)           { return nil, errors.New("not implemented") }

func (c *stubConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if c.connector.expectedQuery != "" && query != c.connector.expectedQuery {
		c.connector.t.Fatalf("unexpected query: %s", query)
	}

	return &stubRows{columns: c.connector.columns, rows: c.connector.rows}, nil
}

func (c *stubConn) Query(query string, args []driver.Value) (driver.Rows, error) {
	return nil, errors.New("not implemented")
}

type stubRows struct {
	columns []string
	rows    [][]driver.Value
	index   int
}

func (r *stubRows) Columns() []string {
	return append([]string(nil), r.columns...)
}

func (r *stubRows) Close() error { return nil }

func (r *stubRows) Next(dest []driver.Value) error {
	if r.index >= len(r.rows) {
		return io.EOF
	}

	copy(dest, r.rows[r.index])
	r.index++

	return nil
}

func newStubDB(t *testing.T, expectedQuery string, columns []string, rows [][]driver.Value) *sql.DB {
	t.Helper()

	connector := &stubConnector{
		t:             t,
		expectedQuery: expectedQuery,
		columns:       columns,
		rows:          rows,
	}

	return sql.OpenDB(connector)
}
