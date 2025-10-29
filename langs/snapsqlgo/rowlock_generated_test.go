package snapsqlgo_test

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"

	querylog "github.com/shibukawa/snapsql/examples/kanban/querylogtest"
	pgquery "github.com/shibukawa/snapsql/examples/kanban/querylogtest/pg"
	snapsqlgo "github.com/shibukawa/snapsql/langs/snapsqlgo"
	"github.com/stretchr/testify/require"
)

type captureSink struct {
	mu      sync.Mutex
	entries []snapsqlgo.QueryLogEntry
}

func (s *captureSink) logger() snapsqlgo.LoggerFunc {
	return func(_ context.Context, entry snapsqlgo.QueryLogEntry) {
		s.mu.Lock()
		defer s.mu.Unlock()

		s.entries = append(s.entries, entry)
	}
}

func (s *captureSink) snapshot() []snapsqlgo.QueryLogEntry {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]snapsqlgo.QueryLogEntry, len(s.entries))
	copy(out, s.entries)

	return out
}

type failingExecutor struct{}

func (f failingExecutor) PrepareContext(context.Context, string) (*sql.Stmt, error) {
	return nil, errors.New("stub prepare")
}

func (f failingExecutor) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, errors.New("stub query")
}

func (f failingExecutor) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return nil, errors.New("stub exec")
}

func TestSQLiteGeneratedSelectRecordsRowLockMode(t *testing.T) {
	sink := &captureSink{}
	ctx := snapsqlgo.WithLogger(context.Background(), sink.logger())
	ctx = snapsqlgo.WithRowLock(ctx)

	_, _ = querylog.BoardGet(ctx, failingExecutor{}, 1)

	entries := sink.snapshot()
	require.Len(t, entries, 1)

	entry := entries[0]
	require.Equal(t, snapsqlgo.RowLockForUpdate, entry.Options.RowLockMode)
	require.Empty(t, entry.Options.RowLockClause, "SQLite dialect should ignore FOR clauses")
	require.NotContains(t, entry.SQL, "FOR UPDATE")
}

func TestSQLiteGeneratedUpdatePanicsWithRowLock(t *testing.T) {
	ctx := snapsqlgo.WithRowLock(context.Background())
	ec := snapsqlgo.ExtractExecutionContext(ctx)
	require.NotNil(t, ec)
	require.Equal(t, snapsqlgo.RowLockForUpdate, ec.RowLockMode())

	require.Panics(t, func() {
		seq := querylog.CardUpdate(ctx, failingExecutor{}, 1, "title", "desc")
		seq(func(*querylog.CardUpdateResult, error) bool { return false })
	})
}

func TestPostgresGeneratedSelectRowLockModes(t *testing.T) {
	cases := []struct {
		name     string
		mode     snapsqlgo.RowLockMode
		expected string
	}{
		{name: "ForUpdate", mode: snapsqlgo.RowLockForUpdate, expected: " FOR UPDATE"},
		{name: "ForShare", mode: snapsqlgo.RowLockForShare, expected: " FOR SHARE"},
		{name: "ForUpdateNoWait", mode: snapsqlgo.RowLockForUpdateNoWait, expected: " FOR UPDATE NOWAIT"},
		{name: "ForUpdateSkipLocked", mode: snapsqlgo.RowLockForUpdateSkipLocked, expected: " FOR UPDATE SKIP LOCKED"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sink := &captureSink{}
			ctx := snapsqlgo.WithLogger(context.Background(), sink.logger())
			ctx = snapsqlgo.WithRowLock(ctx, tc.mode)
			ec := snapsqlgo.ExtractExecutionContext(ctx)
			require.NotNil(t, ec)
			require.Equal(t, tc.mode, ec.RowLockMode())

			_, _ = pgquery.BoardGet(ctx, failingExecutor{}, 1)

			entries := sink.snapshot()
			require.Len(t, entries, 1)

			entry := entries[0]
			require.Equal(t, tc.mode, entry.Options.RowLockMode)
			require.Equal(t, tc.expected, entry.Options.RowLockClause)
			require.Contains(t, entry.SQL, strings.TrimSpace(tc.expected))
		})
	}
}

func TestEnsureRowLockAllowedPanicsForExec(t *testing.T) {
	require.PanicsWithValue(t, "snapsqlgo: WithRowLock is only supported for SELECT queries", func() {
		snapsqlgo.EnsureRowLockAllowed(snapsqlgo.QueryLogQueryTypeExec, snapsqlgo.RowLockForUpdate)
	})
}
