package snapsqlgo_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	snapsqlgo "github.com/shibukawa/snapsql/langs/snapsqlgo"
	generator "github.com/shibukawa/snapsql/testdata/gosample/generated_sqlite"
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

	seq := generator.AccountGet(ctx, noopExecutor{TB: t}, 10)

	var captured []generator.AccountGetResult

	seq(func(res *generator.AccountGetResult, err error) bool {
		require.NoError(t, err)
		require.NotNil(t, res)
		captured = append(captured, *res)

		return false
	})

	require.Len(t, captured, 1)

	if id, ok := captured[0].ID.(float64); ok {
		require.Equal(t, float64(10), id)
	}
}

func TestWithMockProviderAccountUpdate(t *testing.T) {
	provider := newStaticProvider()
	ctx, err := snapsqlgo.WithMockProvider(context.Background(), "AccountUpdate", provider, snapsqlgo.MockOpt{Name: "Update returning", NoRepeat: true})
	require.NoError(t, err)

	rows, err := generator.AccountUpdate(ctx, noopExecutor{TB: t}, 1, "archived")
	require.NoError(t, err)
	require.Len(t, rows, 2)

	require.Equal(t, int64(1), rows[0].ID)
	require.Equal(t, "archived", rows[0].Status)
	require.Equal(t, int64(2), rows[1].ID)
	require.Equal(t, "active", rows[1].Status)

	_, err = generator.AccountUpdate(ctx, noopExecutor{TB: t}, 1, "archived")
	require.ErrorIs(t, err, snapsqlgo.ErrMockSequenceDepleted)
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

	seq := generator.AccountGet(ctx, noopExecutor{TB: t}, 10)
	seq(func(res *generator.AccountGetResult, err error) bool {
		require.ErrorIs(t, err, boom)
		return false
	})

	seq = generator.AccountGet(ctx, noopExecutor{TB: t}, 10)
	seq(func(res *generator.AccountGetResult, err error) bool {
		require.ErrorIs(t, err, snapsqlgo.ErrMockSequenceDepleted)
		return false
	})
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

	if id, ok := captured[0].ID.(float64); ok {
		require.Equal(t, float64(2), id)
	}

	if captured[0].Name != nil {
		if name, ok := (*captured[0].Name).(string); ok {
			require.Equal(t, "Beta", name)
		}
	}

	if id, ok := captured[1].ID.(float64); ok {
		require.Equal(t, float64(1), id)
	}

	iterator = generator.AccountList(ctx, noopExecutor{TB: t})

	var seqErr error

	iterator(func(res *generator.AccountListResult, err error) bool {
		seqErr = err
		return false
	})
	require.ErrorIs(t, seqErr, snapsqlgo.ErrMockSequenceDepleted)
}

func TestFilesystemMockProvider(t *testing.T) {
	startDir, err := filepath.Abs(filepath.Join("..", "..", "testdata", "gosample"))
	require.NoError(t, err)
	require.DirExists(t, filepath.Join(startDir, "testdata", "mock"))
	provider, err := snapsqlgo.NewFilesystemMockProvider(startDir)
	require.NoError(t, err)

	ctx, err := snapsqlgo.WithMockProvider(context.Background(), "AccountGet", provider, snapsqlgo.MockOpt{Name: "Fetch account"})
	require.NoError(t, err)

	seq := generator.AccountGet(ctx, noopExecutor{TB: t}, 10)
	count := 0

	seq(func(res *generator.AccountGetResult, err error) bool {
		require.NoError(t, err)
		require.NotNil(t, res)

		count++

		return false
	})
	require.Equal(t, 1, count)
}

func ptrInt64(v int64) *int64 {
	return &v
}
