package querylogtest

import (
	"context"

	"github.com/shibukawa/snapsql/examples/kanban/internal/query"
	"github.com/shibukawa/snapsql/langs/snapsqlgo"
)

// BoardGetResult exposes the generated result type for testing purposes.
type BoardGetResult = query.BoardGetResult
type CardUpdateResult = query.CardUpdateResult
type BoardListResult = query.BoardListResult

// BoardGet proxies to the generated query.BoardGet to allow external tests to invoke it.
func BoardGet(ctx context.Context, executor snapsqlgo.DBExecutor, boardID int, opts ...snapsqlgo.FuncOpt) (BoardGetResult, error) {
	return query.BoardGet(ctx, executor, boardID, opts...)
}

// CardUpdate proxies to the generated query.CardUpdate for iterator-based verification.
func CardUpdate(ctx context.Context, executor snapsqlgo.DBExecutor, cardID int, title, description string, opts ...snapsqlgo.FuncOpt) func(func(*CardUpdateResult, error) bool) {
	return query.CardUpdate(ctx, executor, cardID, title, description, opts...)
}

// CardPostpone proxies to the generated query.CardPostpone to expose exec-oriented behaviour.
func CardPostpone(ctx context.Context, executor snapsqlgo.DBExecutor, srcBoardID int, dstBoardID int, opts ...snapsqlgo.FuncOpt) (any, error) {
	return query.CardPostpone(ctx, executor, srcBoardID, dstBoardID, opts...)
}

// BoardList proxies to the generated query.BoardList for iterator scenarios.
func BoardList(ctx context.Context, executor snapsqlgo.DBExecutor, opts ...snapsqlgo.FuncOpt) func(func(*BoardListResult, error) bool) {
	return query.BoardList(ctx, executor, opts...)
}
