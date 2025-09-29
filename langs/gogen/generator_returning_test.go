package gogen

import (
	"bytes"
	"encoding/json"
	"os"
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

func TestProcessResponseStructBoardCreateIntermediate(t *testing.T) {
	data, err := os.ReadFile("../../examples/kanban/generated/board_create.json")
	if err != nil {
		t.Fatalf("failed to read intermediate file: %v", err)
	}

	var format intermediate.IntermediateFormat
	if err := json.Unmarshal(data, &format); err != nil {
		t.Fatalf("failed to unmarshal intermediate: %v", err)
	}

	responseType, err := processResponseType(&format)
	if err != nil {
		t.Fatalf("processResponseType returned error: %v", err)
	}

	if responseType == "sql.Result" {
		t.Fatalf("expected typed response, got sql.Result")
	}

	respStruct, err := processResponseStruct(&format)
	if err != nil {
		t.Fatalf("processResponseStruct returned error: %v", err)
	}

	if respStruct == nil {
		t.Fatalf("expected response struct, got nil")
	}

	if len(respStruct.Fields) == 0 {
		t.Fatalf("expected fields in response struct")
	}
}

func TestGenerator_BoardCreateIntermediateGeneratesTypedResult(t *testing.T) {
	data, err := os.ReadFile("../../examples/kanban/generated/board_create.json")
	if err != nil {
		t.Fatalf("failed to read intermediate file: %v", err)
	}

	var format intermediate.IntermediateFormat
	if err := json.Unmarshal(data, &format); err != nil {
		t.Fatalf("failed to unmarshal intermediate: %v", err)
	}

	gen := New(&format, WithPackageName("query"), WithDialect("sqlite"))

	var buf bytes.Buffer
	if err := gen.Generate(&buf); err != nil {
		t.Fatalf("generator failed: %v", err)
	}

	generated := buf.String()
	if !strings.Contains(generated, "type BoardCreateResult struct") {
		t.Fatalf("expected BoardCreateResult struct in generated code, got: %s", generated)
	}

	if strings.Contains(generated, "sql.Result") {
		t.Fatalf("expected generated code to avoid sql.Result, got: %s", generated)
	}
}
