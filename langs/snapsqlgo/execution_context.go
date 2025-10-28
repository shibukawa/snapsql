package snapsqlgo

import "context"

// executionContextKey is the context key used to store ExecutionContext
var executionContextKey struct{}

// ExecutionContext aggregates per-request runtime options that affect generated code execution.
type ExecutionContext struct {
	Logging *LoggingConfig
}

// ExecutionContextOpt mutates an ExecutionContext copy before storing it in context.
type ExecutionContextOpt func(*ExecutionContext)

// WithExecutionContext returns a new context that includes the provided ExecutionContext options.
func WithExecutionContext(ctx context.Context, opts ...ExecutionContextOpt) context.Context {
	existing := ExecutionContextFrom(ctx)
	clone := existing.clone()

	for _, opt := range opts {
		if opt == nil {
			continue
		}

		opt(clone)
	}

	return context.WithValue(ctx, executionContextKey, clone)
}

// ExecutionContextFrom retrieves the aggregated ExecutionContext from context. It never returns nil.
func ExecutionContextFrom(ctx context.Context) *ExecutionContext {
	if ctx == nil {
		return &ExecutionContext{}
	}

	if value := ctx.Value(executionContextKey); value != nil {
		if ec, ok := value.(*ExecutionContext); ok && ec != nil {
			return ec
		}
	}

	return &ExecutionContext{}
}

// WithQueryLogging configures per-request query logging.
func WithQueryLogging(cfg LoggingConfig) ExecutionContextOpt {
	return func(ec *ExecutionContext) {
		copyCfg := cfg
		if copyCfg.IncludeStack && copyCfg.StackDepth <= 0 {
			copyCfg.StackDepth = 16
		}

		if copyCfg.ExplainSlowQueryThreshold < 0 {
			copyCfg.ExplainSlowQueryThreshold = 0
		}

		ec.Logging = &copyCfg
	}
}

// WithLogger is a convenience wrapper that stores logging configuration on the context.
func WithLogger(ctx context.Context, cfg LoggingConfig) context.Context {
	return WithExecutionContext(ctx, WithQueryLogging(cfg))
}

func (ec *ExecutionContext) clone() *ExecutionContext {
	if ec == nil {
		return &ExecutionContext{}
	}

	clone := *ec

	if ec.Logging != nil {
		cfgCopy := *ec.Logging
		clone.Logging = &cfgCopy
	}

	return &clone
}
