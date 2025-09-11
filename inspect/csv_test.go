package inspect

import (
	"strings"
	"testing"
)

func TestTablesCSV_Simple(t *testing.T) {
	sql := "SELECT u.id FROM users u;"

	res, err := Inspect(strings.NewReader(sql), InspectOptions{InspectMode: true})
	if err != nil {
		t.Fatalf("inspect error: %v", err)
	}

	csvb, err := TablesCSV(res, true)
	if err != nil {
		t.Fatalf("csv error: %v", err)
	}

	got := string(csvb)
	// Windows or Unix newline differences avoided by checking contains
	if !strings.Contains(got, "name,alias,schema,source,joinType") {
		t.Fatalf("missing header: %q", got)
	}

	if !strings.Contains(got, "users,u,,main,none") && !strings.Contains(got, "users,,,") {
		t.Fatalf("unexpected row: %q", got)
	}
}
