package snapsqlgo

import (
	"context"
	"testing"
)

type testSink struct {
	entries []QueryLogEntry
}

func (s *testSink) asFunc() QueryLogSink {
	return func(_ context.Context, entry QueryLogEntry) {
		s.entries = append(s.entries, entry)
	}
}

func TestQueryLoggerDisabled(t *testing.T) {
	t.Helper()

	logger := QueryLoggerFromContext(t.Context(), QueryLogMetadata{})
	if logger != nil {
		t.Fatalf("expected nil logger when disabled")
	}
}

func TestQueryLoggerEmitsEntry(t *testing.T) {
	sink := &testSink{}
	ctx := WithLogger(t.Context(), LoggingConfig{Enabled: true, Sink: sink.asFunc()})
	meta := QueryLogMetadata{
		FuncName:   "TestFunc",
		SourceFile: "pkg/TestFunc",
		Dialect:    "postgres",
		QueryType:  QueryLogQueryTypeSelect,
	}

	logger := QueryLoggerFromContext(ctx, meta)
	if logger == nil {
		t.Fatalf("expected logger instance")
	}

	logger.SetQuery("SELECT 1", []any{42})
	logger.Finish(QueryLogExecutionInfo{QueryType: QueryLogQueryTypeSelect}, nil)

	if len(sink.entries) != 1 {
		t.Fatalf("expected one log entry, got %d", len(sink.entries))
	}

	entry := sink.entries[0]
	if entry.SQL != "SELECT 1" {
		t.Errorf("unexpected SQL: %s", entry.SQL)
	}

	if len(entry.Args) != 1 || entry.Args[0] != 42 {
		t.Errorf("unexpected args: %#v", entry.Args)
	}

	if entry.FuncName != "TestFunc" || entry.Dialect != "postgres" {
		t.Errorf("unexpected metadata: %+v", entry)
	}
}

func TestWithExecutionContextCopiesLoggingConfig(t *testing.T) {
	cfg := LoggingConfig{Enabled: true}
	ctx := WithLogger(t.Context(), cfg)

	ec := ExecutionContextFrom(ctx)
	if ec.Logging == nil || !ec.Logging.Enabled {
		t.Fatalf("expected logging config to be set")
	}

	cfg.Enabled = false
	if !ExecutionContextFrom(ctx).Logging.Enabled {
		t.Fatalf("expected stored config to be immutable copy")
	}
}
