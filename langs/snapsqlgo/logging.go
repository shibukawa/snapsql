package snapsqlgo

import (
	"context"
	"database/sql"
	"runtime"
	"strings"
	"time"
)

// LoggingConfig controls query logging behavior.
type LoggingConfig struct {
	Enabled                   bool
	Sink                      QueryLogSink
	IncludeStack              bool
	StackDepth                int
	ExplainMode               ExplainMode
	ExplainSlowQueryThreshold time.Duration
	CaptureParams             bool
	CaptureSQLSource          bool
}

// ExplainMode defines how EXPLAIN should be executed.
type ExplainMode int

const (
	ExplainModeNone ExplainMode = iota
	ExplainModePlan
	ExplainModeAnalyze
)

// QueryLogSink receives QueryLogEntry events.
type QueryLogSink func(context.Context, QueryLogEntry)

// QueryLogQueryType categorizes queries for logging.
type QueryLogQueryType string

const (
	QueryLogQueryTypeSelect QueryLogQueryType = "select"
	QueryLogQueryTypeExec   QueryLogQueryType = "exec"
)

// QueryOptionsSnapshot captures runtime options relevant to logging (placeholder for future RowLock data).
type QueryOptionsSnapshot struct {
	RowLockClause string
}

// QueryLogEntry represents a single query execution event.
type QueryLogEntry struct {
	FuncName     string
	SourceFile   string
	SQL          string
	Args         []any
	Dialect      string
	StartAt      time.Time
	EndAt        time.Time
	Duration     time.Duration
	RowsAffected *int64
	RowCount     *int
	Options      QueryOptionsSnapshot
	StackTrace   []runtime.Frame
	Explain      *ExplainResult
	Error        string
}

// QueryLogMetadata describes immutable attributes passed to the QueryLogger.
type QueryLogMetadata struct {
	FuncName   string
	SourceFile string
	Dialect    string
	QueryType  QueryLogQueryType
	Options    QueryOptionsSnapshot
}

// QueryLogExecutionInfo captures information that emerges only after execution.
type QueryLogExecutionInfo struct {
	Executor        DBExecutor
	QueryType       QueryLogQueryType
	RowsAffected    int64
	HasRowsAffected bool
	RowCount        int
	HasRowCount     bool
}

// QueryLogger coordinates per-query logging lifecycle.
type QueryLogger struct {
	ctx   context.Context
	cfg   *LoggingConfig
	meta  QueryLogMetadata
	entry QueryLogEntry
}

// QueryLoggerFromContext creates a logger using configuration stored on the context.
func QueryLoggerFromContext(ctx context.Context, meta QueryLogMetadata) *QueryLogger {
	execCtx := ExecutionContextFrom(ctx)

	cfg := execCtx.Logging
	if cfg == nil || !cfg.Enabled || cfg.Sink == nil {
		return nil
	}

	logger := &QueryLogger{
		ctx:  ctx,
		cfg:  cfg,
		meta: meta,
		entry: QueryLogEntry{
			FuncName:   meta.FuncName,
			SourceFile: meta.SourceFile,
			Dialect:    meta.Dialect,
			Options:    meta.Options,
			StartAt:    time.Now(),
		},
	}

	return logger
}

// SetQuery captures the SQL text and arguments to be logged.
func (l *QueryLogger) SetQuery(sql string, args []any) {
	if l == nil {
		return
	}

	l.entry.SQL = sql
	if len(args) == 0 {
		l.entry.Args = nil
		return
	}

	copied := make([]any, len(args))
	copy(copied, args)
	l.entry.Args = copied
}

// Finish finalizes the log entry and emits it.
func (l *QueryLogger) Finish(info QueryLogExecutionInfo, err error) {
	if l == nil {
		return
	}

	l.entry.EndAt = time.Now()
	l.entry.Duration = l.entry.EndAt.Sub(l.entry.StartAt)

	if info.HasRowsAffected {
		ra := info.RowsAffected
		l.entry.RowsAffected = &ra
	}

	if info.HasRowCount {
		rc := info.RowCount
		l.entry.RowCount = &rc
	}

	if err != nil {
		l.entry.Error = err.Error()
	}

	if l.cfg.IncludeStack {
		l.entry.StackTrace = captureStackTrace(l.cfg.StackDepth)
	}

	if err == nil && info.QueryType == QueryLogQueryTypeSelect && l.shouldCaptureExplain(l.entry.Duration) {
		if info.Executor != nil {
			if explain := l.runExplain(info.Executor); explain != nil {
				l.entry.Explain = explain
			}
		}
	}

	l.cfg.Sink(l.ctx, l.entry)
}

func (l *QueryLogger) shouldCaptureExplain(duration time.Duration) bool {
	if l.cfg.ExplainMode == ExplainModeNone {
		return false
	}

	threshold := l.cfg.ExplainSlowQueryThreshold
	if threshold > 0 && duration < threshold {
		return false
	}

	return true
}

func (l *QueryLogger) runExplain(executor DBExecutor) *ExplainResult {
	if l.entry.SQL == "" {
		return nil
	}

	query := buildExplainSQL(l.cfg.ExplainMode, l.entry.SQL)

	rows, err := executor.QueryContext(l.ctx, query, l.entry.Args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil
	}

	raw := make([]sql.RawBytes, len(columns))

	dests := make([]any, len(columns))
	for i := range raw {
		dests[i] = &raw[i]
	}

	var planLines []string

	for rows.Next() {
		if err := rows.Scan(dests...); err != nil {
			return nil
		}

		var builder strings.Builder

		for idx, col := range raw {
			if idx > 0 {
				builder.WriteByte('\t')
			}

			if col == nil {
				builder.WriteString("NULL")
				continue
			}

			builder.Write(col)
		}

		planLines = append(planLines, builder.String())
	}

	if len(planLines) == 0 {
		return nil
	}

	return &ExplainResult{QueryPlan: strings.Join(planLines, "\n")}
}

func buildExplainSQL(mode ExplainMode, query string) string {
	prefix := "EXPLAIN "
	if mode == ExplainModeAnalyze {
		prefix = "EXPLAIN ANALYZE "
	}

	return prefix + query
}

func captureStackTrace(depth int) []runtime.Frame {
	if depth <= 0 {
		depth = 16
	}

	pcs := make([]uintptr, depth)
	n := runtime.Callers(3, pcs)
	frames := runtime.CallersFrames(pcs[:n])

	var result []runtime.Frame

	for {
		frame, more := frames.Next()
		result = append(result, frame)

		if !more {
			break
		}
	}

	return result
}
