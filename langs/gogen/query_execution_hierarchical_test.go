package gogen

import (
	"strings"
	"testing"

	"github.com/shibukawa/snapsql/intermediate"
)

// Minimal responseStructData to exercise generateAggregatedScanCode multi-level output
func TestGenerateAggregatedScanCode_MultiLevel(t *testing.T) {
	rs := &responseStructData{
		Name: "SampleQueryResult",
		RawResponses: []intermediate.Response{
			{Name: "id", Type: "int", HierarchyKeyLevel: 1},
			{Name: "a__id", Type: "int", HierarchyKeyLevel: 2},
			{Name: "a__name", Type: "string"},
			{Name: "a__b__id", Type: "int", HierarchyKeyLevel: 3},
			{Name: "a__b__value", Type: "string"},
			{Name: "a__b__c__id", Type: "int", HierarchyKeyLevel: 4},
			{Name: "a__b__c__flag", Type: "bool"},
		},
		// Fields mimic what generateHierarchicalStructs would create (simplified for this test)
		Fields: []responseFieldData{
			{Name: "Id", JSONTag: "id", Type: "int"},
			{Name: "A", JSONTag: "a", Type: "[]*SampleQueryResultA"},
		},
	}

	code, err := generateAggregatedScanCode(rs, true)
	if err != nil {
		t.Fatalf("generateAggregatedScanCode error: %v", err)
	}

	joined := strings.Join(code, "\n")
	// Assertions: ensure maps for each depth, parent map, and chain key usage
	expectedSnippets := []string{
		"var _parentMap map[string]*SampleQueryResult",
		"_nodeMapSampleQueryResultA",   // node map for A
		"_nodeMapSampleQueryResultAB",  // node map for AB
		"_nodeMapSampleQueryResultABC", // node map for ABC
		"parentKey := fmt.Sprintf",
		"for rows.Next()",
		"parentObj.A = append(parentObj.A, node_a)",
	}
	for _, snip := range expectedSnippets {
		if !strings.Contains(joined, snip) {
			t.Errorf("generated code missing snippet: %s", snip)
		}
	}
}
