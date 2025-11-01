package snapsqlgo

import (
	"context"
	"errors"
	"testing"
)

func TestEnforceNonEmptyWhereClause_AllowsWithWhere(t *testing.T) {
	ctx := context.Background()

	meta := &WhereClauseMeta{Status: WhereClauseStatusExists}

	query := "UPDATE users SET name = 'alice' WHERE id = 1"
	if err := EnforceNonEmptyWhereClause(ctx, "UpdateUser", MutationUpdate, meta, query); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestEnforceNonEmptyWhereClause_ErrorsWhenMissing(t *testing.T) {
	ctx := context.Background()
	meta := &WhereClauseMeta{Status: WhereClauseStatusFullScan}

	query := "UPDATE users SET name = 'alice'"

	err := EnforceNonEmptyWhereClause(ctx, "UpdateUser", MutationUpdate, meta, query)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !errors.Is(err, ErrEmptyWhereClause) {
		t.Fatalf("expected ErrEmptyWhereClause, got %v", err)
	}
}

func TestEnforceNonEmptyWhereClause_OptInAllowsExecution(t *testing.T) {
	ctx := WithAllowingNoWhereOperation(context.Background(), "UpdateUser", AllowNoWhereUpdate)
	meta := &WhereClauseMeta{Status: WhereClauseStatusFullScan}

	query := "UPDATE users SET name = 'alice'"
	if err := EnforceNonEmptyWhereClause(ctx, "UpdateUser", MutationUpdate, meta, query); err != nil {
		t.Fatalf("expected opt-in to allow execution, got %v", err)
	}
}

func TestEnforceNonEmptyWhereClause_IgnoresNestedWhere(t *testing.T) {
	ctx := context.Background()
	meta := &WhereClauseMeta{Status: WhereClauseStatusExists}

	query := "UPDATE orders SET status = 'closed' WHERE EXISTS (SELECT 1 FROM line_items WHERE line_items.order_id = orders.id)"
	if err := EnforceNonEmptyWhereClause(ctx, "UpdateOrder", MutationUpdate, meta, query); err != nil {
		t.Fatalf("expected nested WHERE to be ignored, got %v", err)
	}
}

func TestEnforceNonEmptyWhereClause_EmptyAfterKeyword(t *testing.T) {
	ctx := context.Background()
	meta := &WhereClauseMeta{Status: WhereClauseStatusExists}

	query := "UPDATE foo SET bar = 1 WHERE   \n  RETURNING id"

	err := EnforceNonEmptyWhereClause(ctx, "UpdateFoo", MutationUpdate, meta, query)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !errors.Is(err, ErrEmptyWhereClause) {
		t.Fatalf("expected ErrEmptyWhereClause, got %v", err)
	}
}

func TestEnforceNonEmptyWhereClause_ConditionalFallbackTriggered(t *testing.T) {
	ctx := context.Background()
	meta := &WhereClauseMeta{
		Status:            WhereClauseStatusConditional,
		FallbackTriggered: true,
	}

	query := "UPDATE foo SET bar = 1 WHERE 1 = 1"

	err := EnforceNonEmptyWhereClause(ctx, "UpdateFoo", MutationUpdate, meta, query)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !errors.Is(err, ErrEmptyWhereClause) {
		t.Fatalf("expected ErrEmptyWhereClause, got %v", err)
	}
}
