package query

import (
	"strings"
	"testing"
)

// TestDeliverNotificationSQL tests that the generated SQL is correct
func TestDeliverNotificationSQL(t *testing.T) {
	tests := []struct {
		name           string
		userIDs        []string
		wantContains   []string
		wantNoTrailing bool
	}{
		{
			name:           "Single user",
			userIDs:        []string{"USER001"},
			wantContains:   []string{"INSERT INTO inbox", "VALUES", "RETURNING"},
			wantNoTrailing: true,
		},
		{
			name:           "Multiple users",
			userIDs:        []string{"USER001", "USER002", "USER003"},
			wantContains:   []string{"INSERT INTO inbox", "VALUES", "RETURNING"},
			wantNoTrailing: true,
		},
		{
			name:           "Two users",
			userIDs:        []string{"USER001", "USER002"},
			wantContains:   []string{"INSERT INTO inbox", "VALUES", "RETURNING"},
			wantNoTrailing: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := buildDeliverNotificationSQL(tt.userIDs)

			for _, want := range tt.wantContains {
				if !strings.Contains(sql, want) {
					t.Errorf("SQL does not contain %q\nGot: %s", want, sql)
				}
			}

			if tt.wantNoTrailing {
				if strings.Contains(sql, ",  RETURNING") ||
					strings.Contains(sql, ", RETURNING") ||
					strings.Contains(sql, ",RETURNING") {
					t.Errorf("SQL has trailing comma before RETURNING:\n%s", sql)
				}
			}

			valueCount := strings.Count(sql, "($")
			if valueCount != len(tt.userIDs) {
				t.Errorf("Expected %d value tuples, got %d\nSQL: %s", len(tt.userIDs), valueCount, sql)
			}

			t.Logf("Generated SQL:\n%s", sql)
		})
	}
}

func buildDeliverNotificationSQL(userIDs []string) string {
	var builder strings.Builder

	builder.WriteString("INSERT INTO inbox ( notification_id, user_id , created_at, updated_at) VALUES")

	for i := range userIDs {
		isLast := (i == len(userIDs)-1)

		builder.WriteString(" ($1 , $2 , $3 , $4)")

		if !isLast {
			builder.WriteString(" ,")
		}
	}

	builder.WriteString(" RETURNING notification_id, user_id, read_at, created_at")

	return builder.String()
}
