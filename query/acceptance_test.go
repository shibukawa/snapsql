package query

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
	_ "github.com/go-sql-driver/mysql"
	"github.com/goccy/go-yaml"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"

	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// Global containers and DB handles reused across all tests in this package.
var (
	pgOnce    sync.Once
	myOnce    sync.Once
	pgDB      *sql.DB
	myDB      *sql.DB
	pgCont    *postgres.PostgresContainer
	myCont    *mysql.MySQLContainer
	shutdowns []func()
)

func TestMain(m *testing.M) {
	// Run tests; cleanup containers afterwards
	code := m.Run()

	for _, f := range shutdowns {
		f()
	}

	os.Exit(code)
}

func ensurePostgres(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping postgres container in short mode")
	}

	pgOnce.Do(func() {
		cont, err := postgres.Run(t.Context(),
			"postgres:18-alpine",
			postgres.WithDatabase("testdb"),
			postgres.WithUsername("testuser"),
			postgres.WithPassword("testpass"),
			postgres.BasicWaitStrategies(),
		)
		if err != nil {
			t.Fatalf("start postgres: %v", err)
		}

		connStr, err := cont.ConnectionString(t.Context(), "sslmode=disable")
		if err != nil {
			t.Fatalf("postgres conn string: %v", err)
		}

		db, err := sql.Open("pgx", connStr)
		if err != nil {
			t.Fatalf("open postgres: %v", err)
		}

		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(10)

		// wait until ready
		deadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(deadline) {
			if err := db.Ping(); err == nil {
				break
			}

			time.Sleep(500 * time.Millisecond)
		}

		pgCont = cont
		pgDB = db

		shutdowns = append(shutdowns, func() {
			_ = pgDB.Close()
			_ = pgCont.Terminate(t.Context())
		})
	})
}

func ensureMySQL(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping mysql container in short mode")
	}

	myOnce.Do(func() {
		cont, err := mysql.Run(t.Context(),
			"mysql:8.4",
			mysql.WithDatabase("testdb"),
			mysql.WithUsername("testuser"),
			mysql.WithPassword("testpass"),
		)
		if err != nil {
			t.Fatalf("start mysql: %v", err)
		}

		connStr, err := cont.ConnectionString(t.Context())
		if err != nil {
			t.Fatalf("mysql conn string: %v", err)
		}

		db, err := sql.Open("mysql", connStr)
		if err != nil {
			t.Fatalf("open mysql: %v", err)
		}

		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(10)

		// wait until ready
		deadline := time.Now().Add(60 * time.Second)
		for time.Now().Before(deadline) {
			if err := db.Ping(); err == nil {
				break
			}

			time.Sleep(500 * time.Millisecond)
		}

		myCont = cont
		myDB = db

		shutdowns = append(shutdowns, func() {
			_ = myDB.Close()
			_ = myCont.Terminate(t.Context())
		})
	})
}

type expectedResult struct {
	Columns []string         `yaml:"columns,omitempty"`
	Rows    []map[string]any `yaml:"rows"`
	Count   int              `yaml:"count,omitempty"`
}

func toRowMaps(cols []string, rows [][]interface{}) []map[string]any {
	out := make([]map[string]any, len(rows))

	for i, r := range rows {
		m := make(map[string]any, len(cols))

		for j, c := range cols {
			if j < len(r) {
				m[c] = r[j]
			}
		}

		out[i] = m
	}

	return out
}

func TestQueryAcceptance_SQLite(t *testing.T) {
	root := filepath.Join("..", "testdata", "query")

	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		t.Skip("no testdata/query present")
		return
	}

	assert.NoError(t, err)

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		caseDir := filepath.Join(root, e.Name())

		t.Run(e.Name(), func(t *testing.T) {
			// determine input file
			var input string

			for _, name := range []string{"input.snap.sql", "input.snap.md"} {
				p := filepath.Join(caseDir, name)
				if _, err := os.Stat(p); err == nil {
					input = p
					break
				}
			}

			assert.NotEqual(t, "", input, "input file missing")

			// load params if present
			params := map[string]any{}

			for _, fname := range []string{"param.yaml", "params.yaml"} {
				p := filepath.Join(caseDir, fname)
				if b, err := os.ReadFile(p); err == nil {
					_ = yaml.Unmarshal(b, &params)
					break
				}
			}

			// load options if present
			type optionsFile struct {
				Driver         string `yaml:"driver"`
				Limit          int    `yaml:"limit"`
				Offset         int    `yaml:"offset"`
				Explain        bool   `yaml:"explain"`
				ExplainAnalyze bool   `yaml:"explain_analyze"`
			}

			var optf optionsFile
			if b, err := os.ReadFile(filepath.Join(caseDir, "options.yaml")); err == nil {
				_ = yaml.Unmarshal(b, &optf)
			}

			// choose driver and start DB
			drv := strings.ToLower(strings.TrimSpace(optf.Driver))
			if drv == "" {
				drv = "sqlite3"
			}

			var (
				db  *sql.DB
				err error
			)

			switch drv {
			case "sqlite", "sqlite3":
				// SQLite cases are safe to run in parallel
				t.Parallel()
				dbPath := filepath.Join(t.TempDir(), "test.db")
				db, err = OpenDatabase("sqlite3", dbPath, 5)
				assert.NoError(t, err)

				defer db.Close()
			case "postgres", "pgx":
				// Share a single container/DB across tests in this package
				ensurePostgres(t)

				db = pgDB
			case "mysql":
				ensureMySQL(t)

				db = myDB
			default:
				t.Fatalf("unsupported driver: %s", drv)
			}

			// run setup.sql if exists
			if b, err := os.ReadFile(filepath.Join(caseDir, "setup.sql")); err == nil {
				for _, stmt := range splitSQLStatements(string(b)) {
					if stmt == "" {
						continue
					}

					_, err = db.Exec(stmt)
					assert.NoError(t, err)
				}
			}

			// execute
			exec := NewExecutor(db)
			qopt := QueryOptions{Driver: drv, Timeout: 5}
			qopt.Limit = optf.Limit
			qopt.Offset = optf.Offset
			qopt.Explain = optf.Explain || optf.ExplainAnalyze
			qopt.ExplainAnalyze = optf.ExplainAnalyze

			res, err := exec.ExecuteWithTemplate(t.Context(), input, params, qopt)
			assert.NoError(t, err)

			// when explain is on, only check that plan is returned
			if qopt.Explain {
				if res != nil {
					assert.NotEqual(t, "", strings.TrimSpace(res.ExplainPlan))
				}

				return
			}

			// load expected
			b, err := os.ReadFile(filepath.Join(caseDir, "expected.yaml"))
			assert.NoError(t, err)

			var exp expectedResult

			assert.NoError(t, yaml.Unmarshal(b, &exp))

			// compare
			got := expectedResult{Columns: res.Columns, Rows: toRowMaps(res.Columns, res.Rows), Count: res.Count}
			// If exp.Columns omitted, ignore; same for Count
			if len(exp.Columns) == 0 {
				got.Columns = nil
			}

			if exp.Count == 0 {
				got.Count = 0
			}

			// marshal both to YAML for diff-friendly assert
			gy, _ := yaml.Marshal(got)
			ey, _ := yaml.Marshal(exp)
			assert.Equal(t, strings.TrimSpace(string(ey)), strings.TrimSpace(string(gy)))
		})
	}
}

// splitSQLStatements is a very small helper for test setup scripts.
// It splits on semicolons while preserving inner semicolons inside single/double quotes.
// It does not support dollar-quoted PostgreSQL blocks etc., as not needed for current testdata.
func splitSQLStatements(src string) []string {
	var (
		out []string
		b   strings.Builder
	)

	inSingle := false
	inDouble := false

	flush := func() {
		stmt := strings.TrimSpace(b.String())
		if stmt != "" {
			out = append(out, stmt)
		}

		b.Reset()
	}

	for i, r := range src {
		switch r {
		case '\'':
			// toggle single quote if not in double
			if !inDouble {
				inSingle = !inSingle
			}

			b.WriteRune(r)
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}

			b.WriteRune(r)
		case ';':
			if !inSingle && !inDouble {
				flush()
			} else {
				b.WriteRune(r)
			}
		case '\r', '\n':
			// normalize newlines to single space if inside a literal, else keep as whitespace
			if inSingle || inDouble {
				b.WriteRune(' ')
			} else {
				b.WriteRune(r)
			}
		default:
			b.WriteRune(r)
		}

		// If last rune, flush
		if i == len(src)-1 {
			flush()
		}
	}

	return out
}
