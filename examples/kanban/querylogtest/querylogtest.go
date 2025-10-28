package querylogtest

import (
	"context"

	"github.com/shibukawa/snapsql/examples/kanban/internal/query"
	"github.com/shibukawa/snapsql/langs/snapsqlgo"
)

// BoardGetResult exposes the generated result type for testing purposes.
type BoardGetResult = query.BoardGetResult

// BoardGet proxies to the generated query.BoardGet to allow external tests to invoke it.
func BoardGet(ctx context.Context, executor snapsqlgo.DBExecutor, boardID int, opts ...snapsqlgo.FuncOpt) (BoardGetResult, error) {
	return query.BoardGet(ctx, executor, boardID, opts...)
}
