package snapsqlgo

import "context"

// executionContextKeyType is the type for the context key to avoid collisions
type executionContextKeyType struct{}

// executionContextKey is the context key used to store ExecutionContext
var executionContextKey = executionContextKeyType{}

// RowLockMode represents the pessimistic lock mode requested for SELECT statements.
type RowLockMode int

const (
	// RowLockNone disables pessimistic locking.
	RowLockNone RowLockMode = iota
	// RowLockForUpdate emits "FOR UPDATE".
	RowLockForUpdate
	// RowLockForShare emits "FOR SHARE" (where supported).
	RowLockForShare
	// RowLockForUpdateNoWait emits "FOR UPDATE NOWAIT".
	RowLockForUpdateNoWait
	// RowLockForUpdateSkipLocked emits "FOR UPDATE SKIP LOCKED".
	RowLockForUpdateSkipLocked
)

type rowLockConfig struct {
	mode RowLockMode
}

// ExecutionContext aggregates per-request runtime options that affect generated code execution.
type ExecutionContext struct {
	logger  *loggingConfig
	rowLock *rowLockConfig
	mocks   *mockRegistry
}

// RowLockMode reports the configured pessimistic lock mode, defaulting to RowLockNone.
func (ec *ExecutionContext) RowLockMode() RowLockMode {
	if ec == nil || ec.rowLock == nil {
		return RowLockNone
	}

	return ec.rowLock.mode
}

// withExecutionContext returns a context containing the reusable ExecutionContext instance.
func withExecutionContext(ctx context.Context) (context.Context, *ExecutionContext) {
	var ec *ExecutionContext

	if value := ctx.Value(executionContextKey); value != nil {
		var ok bool

		ec, ok = value.(*ExecutionContext)
		if !ok {
			panic("invalid type stored in context for executionContextKey")
		}
	}

	if ec == nil {
		ec = &ExecutionContext{}
		return context.WithValue(ctx, executionContextKey, ec), ec
	}

	return ctx, ec
}

// ExtractExecutionContext retrieves the aggregated ExecutionContext from context.
func ExtractExecutionContext(ctx context.Context) *ExecutionContext {
	if ctx == nil {
		return nil
	}

	if value := ctx.Value(executionContextKey); value != nil {
		if ec, ok := value.(*ExecutionContext); !ok {
			panic("invalid type stored in context for executionContextKey")
		} else {
			return ec
		}
	}

	return nil
}

// WithLogger is a convenience wrapper that stores logging configuration on the context.
func WithLogger(ctx context.Context, logger LoggerFunc, cfg ...LoggerOpt) context.Context {
	ctx, ec := withExecutionContext(ctx)

	var singleOpt LoggerOpt
	if len(cfg) > 0 {
		singleOpt = cfg[0]
	}

	if singleOpt.IncludeStack && singleOpt.StackDepth <= 0 {
		singleOpt.StackDepth = 16
	}

	if singleOpt.ExplainSlowQueryThreshold < 0 {
		singleOpt.ExplainSlowQueryThreshold = 0
	}

	if logger == nil {
		ec.logger = nil
		return ctx
	}

	ec.logger = &loggingConfig{
		logger:                    logger,
		includeStack:              singleOpt.IncludeStack,
		stackDepth:                singleOpt.StackDepth,
		explainMode:               singleOpt.ExplainMode,
		explainSlowQueryThreshold: singleOpt.ExplainSlowQueryThreshold,
	}

	return ctx
}

// WithRowLock records the requested pessimistic lock mode on the context.
//
// The mode defaults to RowLockForUpdate when omitted. Providing RowLockNone clears
// any previously configured lock mode. When multiple modes are provided the last
// value wins, enabling helper wrappers to override earlier defaults.
func WithRowLock(ctx context.Context, modes ...RowLockMode) context.Context {
	mode := RowLockForUpdate
	if len(modes) > 0 {
		mode = modes[len(modes)-1]
	}

	ctx, ec := withExecutionContext(ctx)

	if mode == RowLockNone {
		ec.rowLock = nil
		return ctx
	}

	ec.rowLock = &rowLockConfig{mode: mode}

	return ctx
}
