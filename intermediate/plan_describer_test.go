package intermediate

import "testing"

func TestDescribePlanTables_CTE(t *testing.T) {
	references := []TableReferenceInfo{
		{Name: "done_stage", Context: "cte"},
		{Name: "lists", TableName: "lists", QueryName: "done_stage", Context: "cte"},
	}

	mapping := BuildTableReferenceMap(references)
	physical := []string{"lists"}

	descriptions := DescribePlanTables([]string{"done_stage"}, mapping, physical)

	if len(descriptions) != 1 {
		t.Fatalf("expected 1 description, got %d", len(descriptions))
	}

	expected := "table 'lists' in 'done_stage'(CTE/subquery)"
	if descriptions[0] != expected {
		t.Fatalf("unexpected description: got %q, want %q", descriptions[0], expected)
	}
}

func TestDescribePlanTables_Unresolved(t *testing.T) {
	references := []TableReferenceInfo{}
	mapping := BuildTableReferenceMap(references)

	descriptions := DescribePlanTables([]string{"mystery"}, mapping, nil)

	expected := []string{"table 'mystery' (physical table unresolved)"}
	if len(descriptions) != len(expected) {
		t.Fatalf("unexpected length: got %d, want %d", len(descriptions), len(expected))
	}

	for i := range expected {
		if descriptions[i] != expected[i] {
			t.Fatalf("unexpected description[%d]: got %q, want %q", i, descriptions[i], expected[i])
		}
	}
}

func TestDescribePlanTables_PhysicalAlias(t *testing.T) {
	references := []TableReferenceInfo{
		{Name: "users", TableName: "users", Context: "main"},
	}

	mapping := BuildTableReferenceMap(references)
	physical := []string{"users"}

	descriptions := DescribePlanTables([]string{"users"}, mapping, physical)

	expected := []string{"table 'users'"}
	if len(descriptions) != len(expected) {
		t.Fatalf("unexpected length: got %d, want %d", len(descriptions), len(expected))
	}

	for i := range expected {
		if descriptions[i] != expected[i] {
			t.Fatalf("unexpected description[%d]: got %q, want %q", i, descriptions[i], expected[i])
		}
	}
}
