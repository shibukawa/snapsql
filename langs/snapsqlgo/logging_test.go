package snapsqlgo

import (
	"context"
	"testing"
)

type testSink struct {
	entries []QueryLogEntry
}

func (s *testSink) sink() QueryLogSink {
	return func(_ context.Context, entry QueryLogEntry) {
		s.entries = append(s.entries, entry)
	}
}

func TestQueryLoggerDisabled(t *testing.T) {
	logger := QueryLoggerFromContext(context.Background())
	if logger != nil {
		t.Fatalf("expected nil logger when not configured")
	}
}

func TestQueryLoggerEmitsEntry(t *testing.T) {
	sink := &testSink{}
	ctx := WithLogger(context.Background(), sink.sink())

	logger := QueryLoggerFromContext(ctx)
	if logger == nil {
		t.Fatalf("expected logger instance")
	}

	logger.SetQuery("SELECT 1", []any{42})
	logger.Write(ctx, func() (QueryLogMetadata, DBExecutor) {
		return QueryLogMetadata{
			FuncName:   "TestFunc",
			SourceFile: "pkg/TestFunc",
			Dialect:    "postgres",
			QueryType:  QueryLogQueryTypeSelect,
		}, nil
	})

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

func TestWithLoggerCopiesOptions(t *testing.T) {
	sink := &testSink{}
	opts := LoggerOpt{IncludeStack: true, StackDepth: 0, ExplainSlowQueryThreshold: -1}
	ctx := WithLogger(context.Background(), sink.sink(), opts)

	ec := ExtractExecutionContext(ctx)
	if ec == nil || ec.logger == nil {
		t.Fatalf("expected logger configuration to be stored")
	}

	if !ec.logger.includeStack {
		t.Fatalf("expected includeStack to be true")
	}

	if ec.logger.stackDepth != 16 {
		t.Fatalf("expected stack depth to default to 16, got %d", ec.logger.stackDepth)
	}

	if ec.logger.explainSlowQueryThreshold != 0 {
		t.Fatalf("expected threshold to clamp at 0, got %s", ec.logger.explainSlowQueryThreshold)
	}
}

func TestLoggerWriteNil(t *testing.T) {
	var logger *QueryLogger
	logger.SetQuery("", nil)
	logger.SetErr(nil)
	logger.Write(context.Background(), func() (QueryLogMetadata, DBExecutor) {
		t.Fatalf("callback should not be invoked for nil logger")
		return QueryLogMetadata{}, nil
	})
}
