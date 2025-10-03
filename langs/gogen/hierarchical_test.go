package gogen

import (
	"testing"

	"github.com/shibukawa/snapsql/intermediate"
)

// Test multi-level hierarchical struct generation (a__b__c pattern)
func TestGenerateHierarchicalStructs_MultiLevel(t *testing.T) {
	responses := []intermediate.Response{
		{Name: "id", Type: "int", IsNullable: false, HierarchyKeyLevel: 1},
		{Name: "a__id", Type: "int", IsNullable: false, HierarchyKeyLevel: 2},
		{Name: "a__name", Type: "string", IsNullable: false},
		{Name: "a__b__id", Type: "int", IsNullable: false, HierarchyKeyLevel: 3},
		{Name: "a__b__value", Type: "string", IsNullable: true},
		{Name: "a__b__c__id", Type: "int", IsNullable: false, HierarchyKeyLevel: 4},
		{Name: "a__b__c__flag", Type: "bool", IsNullable: false},
	}

	nodes, rootFields, err := detectHierarchicalStructure(responses)
	if err != nil {
		t.Fatalf("detectHierarchicalStructure error: %v", err)
	}

	if len(rootFields) != 1 {
		t.Fatalf("expected 1 root field, got %d", len(rootFields))
	}

	structs, mainStruct, err := generateHierarchicalStructs("sample_query", nodes, rootFields)
	if err != nil {
		t.Fatalf("generateHierarchicalStructs error: %v", err)
	}

	if mainStruct == nil {
		t.Fatalf("mainStruct is nil")
	}

	if mainStruct.Name == "" {
		t.Errorf("main struct name empty")
	}
	// Expect nested slices fields: A []*SampleQueryA
	foundA := false

	for _, f := range mainStruct.Fields {
		if f.JSONTag == "a" {
			foundA = true

			if f.Type == "" {
				t.Errorf("field type empty for a")
			}
		}
	}

	if !foundA {
		t.Errorf("top-level slice for 'a' not found in main struct")
	}

	// Expect deeper structs exist (A B C) => at least 3 structs (A,B,C)
	if len(structs) < 3 {
		t.Errorf("expected at least 3 nested structs, got %d", len(structs))
	}
}
