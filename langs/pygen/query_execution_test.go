package pygen

import (
	"strings"
	"testing"

	"github.com/shibukawa/snapsql/intermediate"
)

func TestGenerateNoneAffinityExecution_Postgres(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:     "delete_user",
		ResponseAffinity: "none",
	}

	result, err := generateNoneAffinityExecution(format, "postgres")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	code := result.Code
	if !strings.Contains(code, "await conn.execute(sql, *args)") {
		t.Errorf("expected asyncpg execute call, got: %s", code)
	}

	if !strings.Contains(code, "affected_rows") {
		t.Errorf("expected affected_rows extraction, got: %s", code)
	}

	if !strings.Contains(code, "return affected_rows") {
		t.Errorf("expected return affected_rows, got: %s", code)
	}
}

func TestGenerateNoneAffinityExecution_MySQL(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:     "delete_user",
		ResponseAffinity: "none",
	}

	result, err := generateNoneAffinityExecution(format, "mysql")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	code := result.Code
	if !strings.Contains(code, "await cursor.execute(sql, args)") {
		t.Errorf("expected cursor execute call, got: %s", code)
	}

	if !strings.Contains(code, "return cursor.rowcount") {
		t.Errorf("expected return cursor.rowcount, got: %s", code)
	}
}

func TestGenerateNoneAffinityExecution_SQLite(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:     "delete_user",
		ResponseAffinity: "none",
	}

	result, err := generateNoneAffinityExecution(format, "sqlite")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	code := result.Code
	if !strings.Contains(code, "await cursor.execute(sql, args)") {
		t.Errorf("expected cursor execute call, got: %s", code)
	}

	if !strings.Contains(code, "return cursor.rowcount") {
		t.Errorf("expected return cursor.rowcount, got: %s", code)
	}
}

func TestGenerateOneAffinityExecution_Postgres(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:     "get_user",
		ResponseAffinity: "one",
	}

	responseStruct := &responseStructData{
		ClassName: "GetUserResult",
		Fields: []responseFieldData{
			{Name: "user_id", TypeHint: "int"},
			{Name: "username", TypeHint: "str"},
		},
	}

	result, err := generateOneAffinityExecution(format, responseStruct, "postgres")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	code := result.Code
	if !strings.Contains(code, "await conn.fetchrow(sql, *args)") {
		t.Errorf("expected asyncpg fetchrow call, got: %s", code)
	}

	if !strings.Contains(code, "if row is None:") {
		t.Errorf("expected None check, got: %s", code)
	}

	if !strings.Contains(code, "raise NotFoundError") {
		t.Errorf("expected NotFoundError, got: %s", code)
	}

	if !strings.Contains(code, "GetUserResult(**dict(row))") {
		t.Errorf("expected dataclass instantiation, got: %s", code)
	}
}

func TestGenerateOneAffinityExecution_MySQL(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:     "get_user",
		ResponseAffinity: "one",
	}

	responseStruct := &responseStructData{
		ClassName: "GetUserResult",
		Fields: []responseFieldData{
			{Name: "user_id", TypeHint: "int"},
			{Name: "username", TypeHint: "str"},
		},
	}

	result, err := generateOneAffinityExecution(format, responseStruct, "mysql")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	code := result.Code
	if !strings.Contains(code, "await cursor.execute(sql, args)") {
		t.Errorf("expected cursor execute call, got: %s", code)
	}

	if !strings.Contains(code, "row = await cursor.fetchone()") {
		t.Errorf("expected fetchone call, got: %s", code)
	}

	if !strings.Contains(code, "if row is None:") {
		t.Errorf("expected None check, got: %s", code)
	}

	if !strings.Contains(code, "raise NotFoundError") {
		t.Errorf("expected NotFoundError, got: %s", code)
	}

	if !strings.Contains(code, "GetUserResult(**row)") {
		t.Errorf("expected dataclass instantiation, got: %s", code)
	}
}

func TestGenerateManyAffinityExecution_Postgres(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:     "list_users",
		ResponseAffinity: "many",
	}

	responseStruct := &responseStructData{
		ClassName: "ListUsersResult",
		Fields: []responseFieldData{
			{Name: "user_id", TypeHint: "int"},
			{Name: "username", TypeHint: "str"},
		},
	}

	result, err := generateManyAffinityExecution(format, responseStruct, "postgres")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	code := result.Code
	if !strings.Contains(code, "await conn.fetch(sql, *args)") {
		t.Errorf("expected asyncpg fetch call, got: %s", code)
	}

	if !strings.Contains(code, "for row in rows:") {
		t.Errorf("expected for loop, got: %s", code)
	}

	if !strings.Contains(code, "yield ListUsersResult(**dict(row))") {
		t.Errorf("expected yield with dataclass, got: %s", code)
	}
}

func TestGenerateManyAffinityExecution_MySQL(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:     "list_users",
		ResponseAffinity: "many",
	}

	responseStruct := &responseStructData{
		ClassName: "ListUsersResult",
		Fields: []responseFieldData{
			{Name: "user_id", TypeHint: "int"},
			{Name: "username", TypeHint: "str"},
		},
	}

	result, err := generateManyAffinityExecution(format, responseStruct, "mysql")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	code := result.Code
	if !strings.Contains(code, "await cursor.execute(sql, args)") {
		t.Errorf("expected cursor execute call, got: %s", code)
	}

	if !strings.Contains(code, "async for row in cursor:") {
		t.Errorf("expected async for loop, got: %s", code)
	}

	if !strings.Contains(code, "yield ListUsersResult(**row)") {
		t.Errorf("expected yield with dataclass, got: %s", code)
	}
}

func TestGenerateQueryExecution_InvalidAffinity(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:     "test_func",
		ResponseAffinity: "invalid",
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for invalid affinity")
		}
	}()

	generateQueryExecution(format, nil, "postgres")
}

func TestGenerateQueryExecution_MissingResponseStruct(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		FunctionName:     "get_user",
		ResponseAffinity: "one",
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for missing response struct")
		}
	}()

	generateQueryExecution(format, nil, "postgres")
}
