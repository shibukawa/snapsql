package snapsqlgo

import "context"

// executionContextKeyType is the type for the context key to avoid collisions
type executionContextKeyType struct{}

// executionContextKey is the context key used to store ExecutionContext
var executionContextKey = executionContextKeyType{}

// executionContext aggregates per-request runtime options that affect generated code execution.
type executionContext struct {
	logger *loggingConfig
}

// withExecutionContext returns a new context that includes the provided ExecutionContext options.
func withExecutionContext(ctx context.Context) (context.Context, *executionContext) {
	var ec *executionContext

	if value := ctx.Value(executionContextKey); value != nil {
		var ok bool

		ec, ok = value.(*executionContext)
		if !ok {
			panic("invalid type stored in context for executionContextKey")
		}
	}

	if ec == nil {
		ec = &executionContext{}
		return context.WithValue(ctx, executionContextKey, ec), ec
	}

	return ctx, ec
}

// ExtractExecutionContext retrieves the aggregated ExecutionContext from context.
func ExtractExecutionContext(ctx context.Context) *executionContext {
	if ctx == nil {
		return nil
	}

	if value := ctx.Value(executionContextKey); value != nil {
		if ec, ok := value.(*executionContext); !ok {
			panic("invalid type stored in context for executionContextKey")
		} else {
			return ec
		}
	}

	return nil
}

// WithLogger is a convenience wrapper that stores logging configuration on the context.
func WithLogger(ctx context.Context, sink QueryLogSink, cfg ...LoggerOpt) context.Context {
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

	if sink == nil {
		ec.logger = nil
		return ctx
	}

	ec.logger = &loggingConfig{
		sink:                      sink,
		includeStack:              singleOpt.IncludeStack,
		stackDepth:                singleOpt.StackDepth,
		explainMode:               singleOpt.ExplainMode,
		explainSlowQueryThreshold: singleOpt.ExplainSlowQueryThreshold,
	}

	return ctx
}
