package snapsqlgo_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/shibukawa/snapsql/examples/kanban/querylogtest"
	kanbanmock "github.com/shibukawa/snapsql/examples/kanban/testdata/mock"
	"github.com/shibukawa/snapsql/langs/snapsqlgo"
)

type noopExecutor struct {
	TB testing.TB
}

func (n noopExecutor) PrepareContext(context.Context, string) (*sql.Stmt, error) {
	n.TB.Fatalf("unexpected PrepareContext call")
	return nil, errors.New("unreachable")
}

func (n noopExecutor) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	n.TB.Fatalf("unexpected QueryContext call")
	return nil, errors.New("unreachable")
}

func (n noopExecutor) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	n.TB.Fatalf("unexpected ExecContext call")
	return nil, errors.New("unreachable")
}

func TestWithMockEmbeddedProviderBoardGet(t *testing.T) {
	provider := kanbanmock.Provider()
	ctx, err := snapsqlgo.WithMockProvider(context.Background(), "BoardGet", provider, snapsqlgo.MockOpt{Name: "Fetch single board"})
	require.NoError(t, err)

	result, err := querylogtest.BoardGet(ctx, noopExecutor{TB: t}, 10)
	require.NoError(t, err)
	require.Equal(t, 10, result.ID)
	require.Equal(t, "Backlog", result.Name)
	require.Equal(t, "archived", result.Status)
}

func TestWithMockOptionErrorAndNoRepeat(t *testing.T) {
	provider := kanbanmock.Provider()
	boom := errors.New("intentional mock error")
	ctx, err := snapsqlgo.WithMockProvider(context.Background(), "BoardGet", provider, snapsqlgo.MockOpt{
		Name:     "Fetch single board",
		Err:      boom,
		NoRepeat: true,
	})
	require.NoError(t, err)

	_, err = querylogtest.BoardGet(ctx, noopExecutor{TB: t}, 10)
	require.ErrorIs(t, err, boom)

	_, err = querylogtest.BoardGet(ctx, noopExecutor{TB: t}, 10)
	require.ErrorIs(t, err, snapsqlgo.ErrMockSequenceDepleted)
}

func TestWithMockRowsAffectedResult(t *testing.T) {
	provider := kanbanmock.Provider()
	ctx, err := snapsqlgo.WithMockProvider(context.Background(), "CardPostpone", provider, snapsqlgo.MockOpt{
		Name:         "Move undone cards to new board",
		RowsAffected: 5,
		LastInsertID: 99,
		NoRepeat:     true,
	})
	require.NoError(t, err)

	res, err := querylogtest.CardPostpone(ctx, noopExecutor{TB: t}, 10, 20)
	require.NoError(t, err)

	sqlRes, ok := res.(sql.Result)
	require.True(t, ok, "expected sql.Result from mock execution")

	rows, err := sqlRes.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(5), rows)

	lastID, err := sqlRes.LastInsertId()
	require.NoError(t, err)
	require.Equal(t, int64(99), lastID)

	_, err = querylogtest.CardPostpone(ctx, noopExecutor{TB: t}, 10, 20)
	require.ErrorIs(t, err, snapsqlgo.ErrMockSequenceDepleted)
}

func TestWithMockIteratorSequence(t *testing.T) {
	provider := kanbanmock.Provider()
	ctx, err := snapsqlgo.WithMockProvider(context.Background(), "BoardList", provider, snapsqlgo.MockOpt{Name: "Boards are returned in descending creation order", NoRepeat: true})
	require.NoError(t, err)

	iterator := querylogtest.BoardList(ctx, noopExecutor{TB: t})

	var captured []querylogtest.BoardListResult

	iterator(func(res *querylogtest.BoardListResult, err error) bool {
		require.NoError(t, err)
		require.NotNil(t, res)
		captured = append(captured, *res)

		return true
	})

	require.Len(t, captured, 2)
	require.Equal(t, 2, captured[0].ID)
	require.Equal(t, "Project Beta", captured[0].Name)
	require.Equal(t, 1, captured[1].ID)
	require.Equal(t, "Project Alpha", captured[1].Name)

	iterator = querylogtest.BoardList(ctx, noopExecutor{TB: t})

	var seqErr error

	iterator(func(res *querylogtest.BoardListResult, err error) bool {
		seqErr = err
		return false
	})
	require.ErrorIs(t, seqErr, snapsqlgo.ErrMockSequenceDepleted)
}

func TestFilesystemMockProvider(t *testing.T) {
	startDir, err := filepath.Abs(filepath.Join("..", "..", "examples", "kanban"))
	require.NoError(t, err)
	require.DirExists(t, filepath.Join(startDir, "testdata", "mock"))
	provider, err := snapsqlgo.NewFilesystemMockProvider(startDir)
	require.NoError(t, err)

	ctx, err := snapsqlgo.WithMockProvider(context.Background(), "BoardGet", provider, snapsqlgo.MockOpt{Name: "Fetch single board"})
	require.NoError(t, err)

	res, err := querylogtest.BoardGet(ctx, noopExecutor{TB: t}, 10)
	require.NoError(t, err)
	require.Equal(t, 10, res.ID)
}
