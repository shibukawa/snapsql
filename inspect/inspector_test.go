package inspect

import (
	"strings"
	"testing"
)

func TestInspect_SelectSimple(t *testing.T) {
	sql := "SELECT id FROM users;"

	got, err := Inspect(strings.NewReader(sql), InspectOptions{InspectMode: true})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}

	if got.Statement != "select" {
		t.Fatalf("statement = %q, want %q", got.Statement, "select")
	}

	if len(got.Tables) != 1 {
		t.Fatalf("tables len = %d, want %d", len(got.Tables), 1)
	}

	if got.Tables[0].Name != "users" || got.Tables[0].Source != "main" || got.Tables[0].JoinType != "none" {
		t.Fatalf("tables[0] = %+v, want name=users source=main joinType=none", got.Tables[0])
	}
}
