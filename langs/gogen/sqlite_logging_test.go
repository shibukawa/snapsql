package gogen

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/shibukawa/snapsql/examples/kanban/querylogtest"
	"github.com/shibukawa/snapsql/langs/snapsqlgo"
)

type captureSink struct {
	mu      sync.Mutex
	entries []snapsqlgo.QueryLogEntry
}

func (s *captureSink) asFunc() snapsqlgo.QueryLogSink {
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

func setupKanbanBoardTable(t *testing.T, db *sql.DB) {
	t.Helper()
	schema := `CREATE TABLE boards (
        id INTEGER PRIMARY KEY,
        name TEXT NOT NULL,
        status TEXT NOT NULL,
        archived_at DATETIME,
        created_at DATETIME NOT NULL,
        updated_at DATETIME NOT NULL
    );`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to create boards table: %v", err)
	}
	now := time.Now()
	if _, err := db.Exec(`INSERT INTO boards (id, name, status, archived_at, created_at, updated_at) VALUES (?, ?, ?, NULL, ?, ?)`, 1, "Demo Board", "active", now, now); err != nil {
		t.Fatalf("failed to insert board: %v", err)
	}
}

func TestGeneratedKanbanQueryLogging(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	setupKanbanBoardTable(t, db)

	sink := &captureSink{}
	ctx := snapsqlgo.WithLogger(t.Context(), snapsqlgo.LoggingConfig{
		Enabled:       true,
		Sink:          sink.asFunc(),
		CaptureParams: true,
	})

	result, err := querylogtest.BoardGet(ctx, db, 1)
	if err != nil {
		t.Fatalf("BoardGet returned error: %v", err)
	}
	if result.Name != "Demo Board" {
		t.Fatalf("unexpected board name: %s", result.Name)
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
