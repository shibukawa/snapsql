package explain

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsePlanJSONPostgres(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "postgres_sample.json"))
	if err != nil {
		t.Fatalf("failed to read sample: %v", err)
	}

	nodes, err := ParsePlanJSON("postgres", data)
	if err != nil {
		t.Fatalf("ParsePlanJSON returned error: %v", err)
	}

	if len(nodes) == 0 {
		t.Fatalf("expected nodes")
	}

	root := nodes[0]
	if root.NodeType != "Seq Scan" {
		t.Fatalf("expected Seq Scan, got %s", root.NodeType)
	}

	if len(root.Children) != 1 {
		t.Fatalf("expected one child")
	}
}

func TestParsePlanJSONMySQL(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "mysql_sample.json"))
	if err != nil {
		t.Fatalf("failed to read sample: %v", err)
	}

	nodes, err := ParsePlanJSON("mysql", data)
	if err != nil {
		t.Fatalf("ParsePlanJSON returned error: %v", err)
	}

	if len(nodes) != 1 {
		t.Fatalf("expected single node")
	}

	if len(nodes[0].Children) == 0 {
		t.Fatalf("expected child table node")
	}
}
