package snapsqlgo_test

import (
	"context"
	"testing"

	"github.com/shibukawa/snapsql/langs/snapsqlgo"
	generator "github.com/shibukawa/snapsql/testdata/gosample/generated_sqlite"
	"github.com/stretchr/testify/require"
)

func TestGeneratedWhereGuard_AllowsConditionalWhenFilterTrue(t *testing.T) {
	ctx := context.Background()
	mockCase := snapsqlgo.MockCase{
		Name: "update ok",
		Responses: []snapsqlgo.MockResponse{
			{Result: &snapsqlgo.MockSQLResult{RowsAffected: ptrInt64Value(1)}},
		},
	}
	ctx, err := snapsqlgo.WithMock(ctx, "UpdateAccountStatusConditional", []snapsqlgo.MockCase{mockCase})
	require.NoError(t, err)

	res, err := generator.UpdateAccountStatusConditional(ctx, noopExecutor{TB: t}, "active", 10, true)
	require.NoError(t, err)
	rows, err := res.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(1), rows)
}

func TestGeneratedWhereGuard_ErrorsOnFallbackWithoutOptIn(t *testing.T) {
	ctx := context.Background()
	mockCase := snapsqlgo.MockCase{
		Name:      "update fallback",
		Responses: []snapsqlgo.MockResponse{{Result: &snapsqlgo.MockSQLResult{RowsAffected: ptrInt64Value(1)}}},
	}
	ctx, err := snapsqlgo.WithMock(ctx, "UpdateAccountStatusConditional", []snapsqlgo.MockCase{mockCase})
	require.NoError(t, err)

	_, err = generator.UpdateAccountStatusConditional(ctx, noopExecutor{TB: t}, "active", 10, false)
	require.ErrorIs(t, err, snapsqlgo.ErrEmptyWhereClause)
}

func TestGeneratedWhereGuard_AllowsFallbackWithOptIn(t *testing.T) {
	ctx := context.Background()
	mockCase := snapsqlgo.MockCase{
		Name:      "update fallback ok",
		Responses: []snapsqlgo.MockResponse{{Result: &snapsqlgo.MockSQLResult{RowsAffected: ptrInt64Value(1)}}},
	}
	ctx, err := snapsqlgo.WithMock(ctx, "UpdateAccountStatusConditional", []snapsqlgo.MockCase{mockCase})
	require.NoError(t, err)

	ctx = snapsqlgo.WithAllowingNoWhereOperation(ctx, "UpdateAccountStatusConditional", snapsqlgo.AllowNoWhereUpdate)

	res, err := generator.UpdateAccountStatusConditional(ctx, noopExecutor{TB: t}, "active", 10, false)
	require.NoError(t, err)
	rows, err := res.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(1), rows)
}

func ptrInt64Value(v int64) *int64 {
	return &v
}
