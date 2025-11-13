package snapsqlgo_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	snapsqlgo "github.com/shibukawa/snapsql/langs/snapsqlgo"
	generator "github.com/shibukawa/snapsql/testdata/appsample/generated_sqlite"
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

type staticProvider struct {
	cases map[string][]snapsqlgo.MockCase
}

func (p *staticProvider) Cases(functionName string) ([]snapsqlgo.MockCase, error) {
	if cases, ok := p.cases[functionName]; ok {
		return cases, nil
	}

	return nil, snapsqlgo.ErrMockCaseNotFound
}

func newStaticProvider() *staticProvider {
	return &staticProvider{cases: map[string][]snapsqlgo.MockCase{
		"AccountGet": {
			{
				Name: "Fetch account",
				Responses: []snapsqlgo.MockResponse{{
					Expected: []map[string]any{
						{"id": 10, "name": "Primary", "status": "active"},
					},
				}},
			},
		},
		"AccountUpdate": {
			{
				Name: "Update returning",
				Responses: []snapsqlgo.MockResponse{{
					Expected: []map[string]any{
						{"id": 1, "status": "archived"},
						{"id": 2, "status": "active"},
					},
				}},
			},
		},
		"UpdateAccountStatusConditional": {
			{
				Name: "Update ok",
				Responses: []snapsqlgo.MockResponse{{
					Result: &snapsqlgo.MockSQLResult{RowsAffected: ptrInt64(5), LastInsertID: ptrInt64(99)},
				}},
			},
		},
		"AccountList": {
			{
				Name: "Descending order",
				Responses: []snapsqlgo.MockResponse{{
					Expected: []map[string]any{
						{"id": 2, "name": "Beta", "status": "active"},
						{"id": 1, "name": "Alpha", "status": "archived"},
					},
				}},
			},
		},
	}}
}

func TestWithMockProviderAccountGet(t *testing.T) {
	provider := newStaticProvider()
	ctx, err := snapsqlgo.WithMockProvider(context.Background(), "AccountGet", provider, snapsqlgo.MockOpt{Name: "Fetch account"})
	require.NoError(t, err)

	result, err := generator.AccountGet(ctx, noopExecutor{TB: t}, 10)
	require.NoError(t, err)
	require.Equal(t, 10, result.ID)
	require.NotNil(t, result.Name)
	require.Equal(t, "Primary", *result.Name)
	require.NotNil(t, result.Status)
	require.Equal(t, "active", *result.Status)
}

func TestWithMockProviderAccountUpdate(t *testing.T) {
	provider := newStaticProvider()
	ctx, err := snapsqlgo.WithMockProvider(context.Background(), "AccountUpdate", provider, snapsqlgo.MockOpt{Name: "Update returning", NoRepeat: true})
	require.NoError(t, err)

	iterator := generator.AccountUpdate(ctx, noopExecutor{TB: t}, 1, "archived")

	var captured []generator.AccountUpdateResult

	iterator(func(res *generator.AccountUpdateResult, err error) bool {
		require.NoError(t, err)
		require.NotNil(t, res)
		captured = append(captured, *res)

		return true
	})

	require.Len(t, captured, 2)
	require.Equal(t, 1, captured[0].ID)
	require.NotNil(t, captured[0].Status)
	require.Equal(t, "archived", *captured[0].Status)
	require.Equal(t, 2, captured[1].ID)
	require.NotNil(t, captured[1].Status)
	require.Equal(t, "active", *captured[1].Status)

	iterator = generator.AccountUpdate(ctx, noopExecutor{TB: t}, 1, "archived")
	called := false

	iterator(func(res *generator.AccountUpdateResult, err error) bool {
		called = true

		require.ErrorIs(t, err, snapsqlgo.ErrMockSequenceDepleted)

		return false
	})
	require.True(t, called)
}

func TestWithMockOptionErrorAndNoRepeat(t *testing.T) {
	provider := newStaticProvider()
	boom := errors.New("intentional mock error")
	ctx, err := snapsqlgo.WithMockProvider(context.Background(), "AccountGet", provider, snapsqlgo.MockOpt{
		Name:     "Fetch account",
		Err:      boom,
		NoRepeat: true,
	})
	require.NoError(t, err)

	_, err = generator.AccountGet(ctx, noopExecutor{TB: t}, 10)
	require.ErrorIs(t, err, boom)

	_, err = generator.AccountGet(ctx, noopExecutor{TB: t}, 10)
	require.ErrorIs(t, err, snapsqlgo.ErrMockSequenceDepleted)
}

func TestWithMockRowsAffectedResult(t *testing.T) {
	provider := newStaticProvider()
	ctx, err := snapsqlgo.WithMockProvider(context.Background(), "UpdateAccountStatusConditional", provider, snapsqlgo.MockOpt{
		Name:         "Update ok",
		RowsAffected: 5,
		LastInsertID: 99,
		NoRepeat:     true,
	})
	require.NoError(t, err)

	sqlRes, err := generator.UpdateAccountStatusConditional(ctx, noopExecutor{TB: t}, "active", 10, true)
	require.NoError(t, err)

	rows, err := sqlRes.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(5), rows)

	lastID, err := sqlRes.LastInsertId()
	require.NoError(t, err)
	require.Equal(t, int64(99), lastID)

	_, err = generator.UpdateAccountStatusConditional(ctx, noopExecutor{TB: t}, "active", 10, true)
	require.ErrorIs(t, err, snapsqlgo.ErrMockSequenceDepleted)
}

func TestWithMockIteratorSequence(t *testing.T) {
	provider := newStaticProvider()
	ctx, err := snapsqlgo.WithMockProvider(context.Background(), "AccountList", provider, snapsqlgo.MockOpt{Name: "Descending order", NoRepeat: true})
	require.NoError(t, err)

	iterator := generator.AccountList(ctx, noopExecutor{TB: t})

	var captured []generator.AccountListResult

	iterator(func(res *generator.AccountListResult, err error) bool {
		require.NoError(t, err)
		require.NotNil(t, res)
		captured = append(captured, *res)

		return true
	})

	require.Len(t, captured, 2)
	require.Equal(t, 2, captured[0].ID)
	require.NotNil(t, captured[0].Name)
	require.Equal(t, "Beta", *captured[0].Name)
	require.NotNil(t, captured[0].Status)
	require.Equal(t, "active", *captured[0].Status)
	require.Equal(t, 1, captured[1].ID)
	require.NotNil(t, captured[1].Name)
	require.Equal(t, "Alpha", *captured[1].Name)
	require.NotNil(t, captured[1].Status)
	require.Equal(t, "archived", *captured[1].Status)

	iterator = generator.AccountList(ctx, noopExecutor{TB: t})

	var seqErr error

	iterator(func(res *generator.AccountListResult, err error) bool {
		seqErr = err
		return false
	})
	require.ErrorIs(t, seqErr, snapsqlgo.ErrMockSequenceDepleted)
}

func TestFilesystemMockProvider(t *testing.T) {
	startDir, err := filepath.Abs(filepath.Join("..", "..", "testdata", "appsample"))
	require.NoError(t, err)
	require.DirExists(t, filepath.Join(startDir, "testdata", "mock"))
	provider, err := snapsqlgo.NewFilesystemMockProvider(startDir)
	require.NoError(t, err)

	ctx, err := snapsqlgo.WithMockProvider(context.Background(), "AccountGet", provider, snapsqlgo.MockOpt{Name: "Fetch account"})
	require.NoError(t, err)

	result, err := generator.AccountGet(ctx, noopExecutor{TB: t}, 10)
	require.NoError(t, err)
	require.Equal(t, 10, result.ID)
}

func ptrInt64(v int64) *int64 {
	return &v
}
