package gogen

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shibukawa/snapsql/intermediate"
)

// Test that INSERT ... RETURNING with response_affinity already set to one generates Scan code
func TestGenerator_DMLReturningInsertOneAffinity(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:     "board_create",
		Description:      "Creates a board (test)",
		ResponseAffinity: "one", // pipeline should have promoted earlier
		Responses: []intermediate.Response{
			{Name: "id", Type: "int"},
			{Name: "name", Type: "string"},
		},
		Instructions: []intermediate.Instruction{
			{Op: intermediate.OpEmitStatic, Value: "INSERT INTO boards (name) VALUES ($1) RETURNING id, name"},
		},
	}

	g := New(format, WithPackageName("query"), WithDialect("postgres"))

	var buf bytes.Buffer
	if err := g.Generate(&buf); err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	code := buf.String()
	if !strings.Contains(code, "type BoardCreateResult struct") {
		t.Fatalf("expected result struct to be generated, got code: %s", code)
	}

	if !strings.Contains(code, "QueryRowContext") || !strings.Contains(code, "Scan(") {
		t.Fatalf("expected single row scan with QueryRowContext and Scan, got code: %s", code)
	}
}

// Test that UPDATE ... RETURNING with affinity one generates Scan code
func TestGenerator_DMLReturningUpdateOneAffinity(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:     "card_move",
		ResponseAffinity: "one",
		Responses: []intermediate.Response{
			{Name: "id", Type: "int"},
			{Name: "title", Type: "string"},
		},
		Instructions: []intermediate.Instruction{
			{Op: intermediate.OpEmitStatic, Value: "UPDATE cards SET title=$1 WHERE id=$2 RETURNING id, title"},
		},
	}

	g := New(format, WithPackageName("query"), WithDialect("postgres"))

	var buf bytes.Buffer
	if err := g.Generate(&buf); err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	code := buf.String()
	if !strings.Contains(code, "CardMoveResult") {
		t.Fatalf("expected CardMoveResult struct, got: %s", code)
	}

	if !strings.Contains(code, "QueryRowContext") || !strings.Contains(code, "Scan(") {
		t.Fatalf("expected QueryRowContext scan, got: %s", code)
	}
}
