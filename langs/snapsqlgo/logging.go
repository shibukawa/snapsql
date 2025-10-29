package snapsqlgo

import (
	"context"
	"database/sql"
	"runtime"
	"strings"
	"time"
)

// loggingConfig controls query logging behaviour stored on context.
type loggingConfig struct {
	logger                    LoggerFunc
	includeStack              bool
	stackDepth                int
	explainMode               ExplainMode
	explainSlowQueryThreshold time.Duration
}

// LoggerOpt configures optional logger behaviour passed to WithLogger.
type LoggerOpt struct {
	IncludeStack              bool
	StackDepth                int
	ExplainMode               ExplainMode
	ExplainSlowQueryThreshold time.Duration
}

// ExplainMode defines how EXPLAIN should be executed.
type ExplainMode int

const (
	ExplainModeNone ExplainMode = iota
	ExplainModePlan
	ExplainModeAnalyze
)

// LoggerFunc receives QueryLogEntry events.
type LoggerFunc func(context.Context, QueryLogEntry)

// QueryLogQueryType categorizes queries for logging.
type QueryLogQueryType string

const (
	QueryLogQueryTypeSelect QueryLogQueryType = "select"
	QueryLogQueryTypeExec   QueryLogQueryType = "exec"
)

// QueryOptionsSnapshot captures runtime options relevant to logging.
type QueryOptionsSnapshot struct {
	RowLockClause string
	RowLockMode   RowLockMode
}

// QueryLogEntry represents a single query execution event.
type QueryLogEntry struct {
	FuncName   string
	SourceFile string
	SQL        string
	Args       []any
	Dialect    string
	StartAt    time.Time
	EndAt      time.Time
	Duration   time.Duration
	Options    QueryOptionsSnapshot
	StackTrace []runtime.Frame
	Explain    *ExplainResult
	Error      string
}

// QueryLogMetadata describes immutable attributes passed to the QueryLogger.
type QueryLogMetadata struct {
	FuncName   string
	SourceFile string
	Dialect    string
	QueryType  QueryLogQueryType
	Options    QueryOptionsSnapshot
}

// QueryLogger coordinates per-query logging lifecycle.
type QueryLogger struct {
	cfg     *loggingConfig
	startAt time.Time
	sql     string
	args    []any
	err     error
}

// QueryLogger produces a QueryLogger from the execution context.
func (ec *ExecutionContext) QueryLogger() *QueryLogger {
	if ec == nil || ec.logger == nil || ec.logger.logger == nil {
		return nil
	}

	return &QueryLogger{
		cfg:     ec.logger,
		startAt: time.Now(),
	}
}

// SetQuery captures the SQL text and arguments to be logged.
func (l *QueryLogger) SetQuery(sql string, args []any) {
	if l == nil {
		return
	}

	l.sql = sql
	if len(args) == 0 {
		l.args = nil
		return
	}

	copied := make([]any, len(args))
	copy(copied, args)
	l.args = copied
}

// SetErr records the last error to be logged.
func (l *QueryLogger) SetErr(err error) {
	if l == nil {
		return
	}

	l.err = err
}

// Write finalizes the log entry via the provided metadata callback.
func (l *QueryLogger) Write(ctx context.Context, metaProvider func() (QueryLogMetadata, DBExecutor)) {
	if l == nil {
		return
	}

	metadata, executor := metaProvider()

	entry := QueryLogEntry{
		FuncName:   metadata.FuncName,
		SourceFile: metadata.SourceFile,
		Dialect:    metadata.Dialect,
		Options:    metadata.Options,
		StartAt:    l.startAt,
		EndAt:      time.Now(),
	}
	entry.Duration = entry.EndAt.Sub(entry.StartAt)
	entry.SQL = l.sql

	if len(l.args) > 0 {
		copied := make([]any, len(l.args))
		copy(copied, l.args)
		entry.Args = copied
	}

	if l.err != nil {
		entry.Error = l.err.Error()
	}

	if l.cfg.includeStack {
		entry.StackTrace = captureStackTrace(l.cfg.stackDepth)
	}

	if l.err == nil && metadata.QueryType == QueryLogQueryTypeSelect && l.shouldCaptureExplain(entry.Duration) {
		if executor != nil && entry.SQL != "" {
			if explain := l.runExplain(ctx, executor, entry.SQL, l.args); explain != nil {
				entry.Explain = explain
			}
		}
	}

	l.cfg.logger(ctx, entry)
}

func (l *QueryLogger) shouldCaptureExplain(duration time.Duration) bool {
	if l.cfg.explainMode == ExplainModeNone {
		return false
	}

	threshold := l.cfg.explainSlowQueryThreshold
	if threshold > 0 && duration < threshold {
		return false
	}

	return true
}

func (l *QueryLogger) runExplain(ctx context.Context, executor DBExecutor, query string, args []any) *ExplainResult {
	explainSQL := buildExplainSQL(l.cfg.explainMode, query)

	rows, err := executor.QueryContext(ctx, explainSQL, args...)
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

	if err := rows.Err(); err != nil {
		return nil
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
