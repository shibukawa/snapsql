package testrunner

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/goccy/go-yaml"
	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/markdownparser"
	fr "github.com/shibukawa/snapsql/testrunner/fixtureexecutor"
)

// TestMarkdownAcceptance
// - testdata/testrunner/markdown 配下の ok_ / ng_ / skip_ ファイルを探索
// - dialectごとに1つのDB(コンテナ or :memory:) を起動し __init.sql を適用
// - 各Markdownファイルは t.Run でネストし、Markdown内の各テストケースは Executor 側で BEGIN/ROLLBACK
// - ok_: すべて成功 expected, ng_: 1つ以上の比較失敗 expected, skip_: 無視
// まだ ng_ ケースは未投入でも動作するようにしている
func TestMarkdownAcceptance(t *testing.T) {
	baseDir := filepath.Join("..", "testdata", "testrunner", "markdown")

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		t.Fatalf("read markdown dir: %v", err)
	}

	groups := map[string][]string{}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		name := e.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}

		if strings.HasPrefix(name, "skip_") {
			continue
		}
		// 追加プレフィックス: fail_ (ng_ と同義), err_ (実行エラー期待)
		if !strings.HasPrefix(name, "ok_") && !strings.HasPrefix(name, "ng_") && !strings.HasPrefix(name, "fail_") && !strings.HasPrefix(name, "err_") {
			t.Fatalf("file %s must start with ok_ / ng_ / fail_ / err_ / skip_", name)
		}

		dialect := detectDialect(filepath.Join(baseDir, name))
		if dialect == "" {
			dialect = "sqlite"
		}
		// frontmatterで dialect: "sqlite" のように引用符付きなので除去
		dialect = strings.Trim(dialect, "\"' ")
		groups[dialect] = append(groups[dialect], name)
	}

	for dialect, files := range groups {
		d := dialect
		t.Run("dialect="+d, func(t *testing.T) {
			// Run dialect groups in parallel to overlap container start times
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()

			db, cleanup := startDialectDB(ctx, t, d)
			defer cleanup()

			applyInitSQL(t, db, baseDir)

			for _, name := range files {
				t.Run(name, func(t *testing.T) {
					t.Parallel()
					// Parallelize file-level tests only for non-sqlite dialects to avoid :memory: quirks
					//if d != "sqlite" && d != "sqlite3" {
					//	t.Parallel()
					//}
					path := filepath.Join(baseDir, name)

					f, err := os.Open(path)
					if err != nil {
						t.Fatalf("open %s: %v", name, err)
					}

					defer f.Close()

					doc, err := markdownparser.Parse(f)
					if err != nil {
						if strings.HasPrefix(name, "err_") {
							// パース失敗を実行エラー種別の成功とみなす
							t.Logf("expected parse error for err_ file: %v", err)
							// 後段のサマリ判定用に executionErrorAny=true へ直接ジャンプ
							// 何も実行せずreturnする
							return
						}
						// err_ 以外は致命
						t.Fatalf("parse %s: %v", name, err)
					}

					schemaMap := loadTestSchema(t, baseDir, d)
					exec := fr.NewExecutor(db, d, schemaMap)
					exec.SetBaseDir(baseDir)

					failedAny := false
					executionErrorAny := false

					for i := range doc.TestCases {
						c := &doc.TestCases[i]

						_, err := exec.ExecuteTest(c, doc.SQL, map[string]any{}, &fr.ExecutionOptions{Mode: fr.FullTest, Commit: false})
						if err != nil {
							if strings.Contains(err.Error(), "simple validation failed:") || strings.Contains(err.Error(), "table state validation failed:") {
								failedAny = true

								if strings.HasPrefix(name, "ok_") {
									t.Errorf("case %s validation failed: %v", c.Name, err)
								}
							} else {
								// 実行エラー (SQL構文エラーなど)
								if strings.HasPrefix(name, "err_") {
									executionErrorAny = true
								} else {
									t.Fatalf("execution error: %v", err)
								}
							}
						}
					}

					if strings.HasPrefix(name, "ok_") && failedAny {
						t.Fatalf("ok_ file expected all pass but had failures")
					}

					if (strings.HasPrefix(name, "ng_") || strings.HasPrefix(name, "fail_")) && !failedAny {
						t.Fatalf("ng_ file expected at least one comparison failure but all passed")
					}

					if strings.HasPrefix(name, "err_") && !executionErrorAny {
						t.Fatalf("err_ file expected execution error but all passed")
					}
				})
			}
		})
	}
}

// startDialectDB: dialectごとに1コンテナ(or in-memory)起動
func startDialectDB(ctx context.Context, t *testing.T, dialect string) (*sql.DB, func()) {
	t.Helper()

	switch strings.ToLower(dialect) {
	case "postgres", "postgresql":
		cont, err := postgres.Run(ctx,
			"postgres:17-alpine",
			postgres.WithDatabase("testdb"),
			postgres.WithUsername("testuser"),
			postgres.WithPassword("testpass"),
			postgres.BasicWaitStrategies(),
		)
		if err != nil {
			t.Fatalf("start postgres: %v", err)
		}

		connStr, err := cont.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			t.Fatalf("conn string: %v", err)
		}

		db, err := sql.Open("pgx", connStr)
		if err != nil {
			t.Fatalf("open pg: %v", err)
		}
		// Allow concurrency for parallel subtests
		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(10)

		return db, func() { db.Close(); cont.Terminate(context.Background()) }
	case "mysql":
		cont, err := mysql.Run(ctx,
			"mysql:8.4",
			mysql.WithDatabase("testdb"),
			mysql.WithUsername("testuser"),
			mysql.WithPassword("testpass"),
		)
		if err != nil {
			t.Fatalf("start mysql: %v", err)
		}

		connStr, err := cont.ConnectionString(ctx)
		if err != nil {
			t.Fatalf("conn string: %v", err)
		}

		db, err := sql.Open("mysql", connStr)
		if err != nil {
			t.Fatalf("open mysql: %v", err)
		}
		// Allow concurrency for parallel subtests
		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(10)

		return db, func() { db.Close(); cont.Terminate(context.Background()) }
	case "sqlite":
		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			t.Fatalf("open sqlite: %v", err)
		}
		// Use a single connection for :memory: DB to avoid multiple independent databases
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)

		return db, func() { db.Close() }
	default:
		t.Fatalf("unsupported dialect: %s", dialect)
		return nil, func() {}
	}
}

// applyInitSQL: __init.sql を適用
func applyInitSQL(t *testing.T, db *sql.DB, baseDir string) {
	t.Helper()
	// サブテスト名から dialect=xxx を抽出
	dialect := ""

	if parts := strings.Split(t.Name(), "/"); len(parts) > 1 {
		for _, p := range parts {
			if strings.HasPrefix(p, "dialect=") {
				dialect = strings.TrimPrefix(p, "dialect=")
				break
			}
		}
	}

	if dialect == "" {
		dialect = "sqlite"
	}

	path := filepath.Join(baseDir, "__init_"+dialect+".sql")

	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", filepath.Base(path), err)
	}

	if len(content) == 0 {
		return
	}

	text := string(content)
	if dialect == "mysql" {
		stmts := splitSQLStatements(text)
		for _, stmt := range stmts {
			if strings.TrimSpace(stmt) == "" {
				continue
			}

			if _, err := db.Exec(stmt); err != nil {
				if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "UNIQUE constraint failed") {
					continue
				}

				t.Fatalf("apply %s stmt error: %v", filepath.Base(path), err)
			}
		}

		return
	}

	if _, err := db.Exec(text); err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return
		}

		t.Fatalf("apply %s: %v", filepath.Base(path), err)
	}
}

// splitSQLStatements: セミコロン終端で単純分割。文字列リテラル内のセミコロンや複雑な構文は今回の初期化DDL想定外なので未対応で十分。
func splitSQLStatements(sqlText string) []string {
	parts := strings.Split(sqlText, ";")

	res := make([]string, 0, len(parts))

	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}

		res = append(res, trimmed+";")
	}

	return res
}

// detectDialect: 先頭frontmatterを簡易解析して dialect: を取得
func detectDialect(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	txt := string(b)
	if !strings.HasPrefix(txt, "---") {
		return ""
	}

	rest := txt[3:]

	end := strings.Index(rest, "---")
	if end == -1 {
		return ""
	}

	header := rest[:end]
	for _, line := range strings.Split(header, "\n") {
		raw := strings.TrimSpace(line)

		lower := strings.ToLower(raw)

		if strings.HasPrefix(lower, "dialect:") {
			parts := strings.SplitN(raw, ":", 2)
			if len(parts) == 2 {
				v := strings.TrimSpace(parts[1])
				v = strings.Trim(v, "\"'")

				return v
			}
		}
	}

	return ""
}

// loadTestSchema: __schema_<dialect>.yaml を読み込み tableInfo map を生成
func loadTestSchema(t *testing.T, baseDir, dialect string) map[string]*snapsql.TableInfo {
	t.Helper()
	// 今回は postgres のみ用意。存在しなければ空map
	file := filepath.Join(baseDir, "__schema_"+dialect+".yaml")
	if _, err := os.Stat(file); err != nil {
		return map[string]*snapsql.TableInfo{}
	}

	b, err := os.ReadFile(file)
	if err != nil {
		// 読み込み失敗はテスト失敗
		return map[string]*snapsql.TableInfo{}
	}
	// 想定YAML構造を匿名structで受ける
	var raw struct {
		Tables []struct {
			Name    string `yaml:"name"`
			Columns map[string]struct {
				Name         string `yaml:"name"`
				DataType     string `yaml:"data_type"`
				Nullable     bool   `yaml:"nullable"`
				IsPrimaryKey bool   `yaml:"is_primary_key"`
			} `yaml:"columns"`
			ColumnOrder []string `yaml:"column_order"`
		} `yaml:"tables"`
	}

	if err := yaml.Unmarshal(b, &raw); err != nil {
		return map[string]*snapsql.TableInfo{}
	}

	res := make(map[string]*snapsql.TableInfo)

	for _, tbl := range raw.Tables {
		cols := make(map[string]*snapsql.ColumnInfo)
		for name, c := range tbl.Columns {
			cols[name] = &snapsql.ColumnInfo{
				Name:         c.Name,
				DataType:     c.DataType,
				Nullable:     c.Nullable,
				IsPrimaryKey: c.IsPrimaryKey,
			}
		}

		res[tbl.Name] = &snapsql.TableInfo{Columns: cols, ColumnOrder: tbl.ColumnOrder, Name: tbl.Name}
	}

	return res
}
