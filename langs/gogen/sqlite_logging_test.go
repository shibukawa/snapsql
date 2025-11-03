package gogen

import (
	"context"
	"database/sql"
	"sync"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/shibukawa/snapsql/langs/snapsqlgo"
	generator "github.com/shibukawa/snapsql/testdata/gosample/generated_sqlite"
)

type captureSink struct {
	mu      sync.Mutex
	entries []snapsqlgo.QueryLogEntry
}

func (s *captureSink) asFunc() snapsqlgo.LoggerFunc {
	return func(_ context.Context, entry snapsqlgo.QueryLogEntry) {
		s.mu.Lock()
		defer s.mu.Unlock()

		s.entries = append(s.entries, entry)
	}
}

func (s *captureSink) Entries() []snapsqlgo.QueryLogEntry {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]snapsqlgo.QueryLogEntry, len(s.entries))
	copy(out, s.entries)

	return out
}

func setupAccountsTable(t *testing.T, db *sql.DB) {
	t.Helper()

	schema := `CREATE TABLE accounts (
		id INTEGER PRIMARY KEY,
		name TEXT,
		status TEXT
	);`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to create accounts table: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO accounts (id, name, status) VALUES (?, ?, ?)`, 1, "Demo Account", "active"); err != nil {
		t.Fatalf("failed to insert account: %v", err)
	}
}

func TestGeneratedKanbanQueryLogging(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	setupAccountsTable(t, db)

	sink := &captureSink{}
	ctx := snapsqlgo.WithLogger(context.Background(), sink.asFunc())

	seq := generator.AccountGet(ctx, db, 1)

	var got *generator.AccountGetResult

	seq(func(res *generator.AccountGetResult, err error) bool {
		if err != nil {
			t.Fatalf("AccountGet returned error: %v", err)
		}

		got = res

		return false
	})

	if got == nil {
		t.Fatalf("expected AccountGet to yield a result")
	}

	entries := sink.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.SQL == "" {
		t.Fatalf("expected SQL to be captured")
	}

	if len(entry.Args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(entry.Args))
	}

	if entry.Error != "" {
		t.Fatalf("expected successful log, got error %s", entry.Error)
	}
}
