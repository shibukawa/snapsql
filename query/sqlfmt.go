package query

import (
	"strconv"
	"strings"

	"github.com/shibukawa/snapsql"
)

// FormatSQLForDriver converts placeholders for the given database driver and
// ensures readability by adding a space after placeholders when needed.
// Supported drivers for numbered placeholders: postgres, pgx, postgresql.
func FormatSQLForDriver(sql string, driver string) string {
	return ensureSpaceAfterPlaceholders(sql)
}

// FormatSQLForDialect converts placeholders for display (dry-run use)
// and ensures readability by adding a space after placeholders when needed.
// Dialect: postgresql/mysql/sqlite.
func FormatSQLForDialect(sql string, dialect snapsql.Dialect) string {
	d := strings.ToLower(strings.TrimSpace(string(dialect)))
	switch d {
	case "postgres", "postgresql", "pg", "pgx":
		var b strings.Builder

		n := 1
		inSingle := false
		inDouble := false

		for i := range len(sql) {
			ch := sql[i]
			if ch == '\'' && !inDouble {
				inSingle = !inSingle

				b.WriteByte(ch)

				continue
			}

			if ch == '"' && !inSingle {
				inDouble = !inDouble

				b.WriteByte(ch)

				continue
			}

			if ch == '?' && !inSingle && !inDouble {
				b.WriteByte('$')
				b.WriteString(strconv.Itoa(n))

				n++

				continue
			}

			b.WriteByte(ch)
		}

		return ensureSpaceAfterPlaceholders(b.String())
	default:
		return ensureSpaceAfterPlaceholders(sql)
	}
}

// ensureSpaceAfterPlaceholders inserts a single space after a placeholder token
// when the next character is an identifier character (letter, number, underscore).
// Supported placeholders: '?' and PostgreSQL-style '$<digits>'. Quotes are respected.
func ensureSpaceAfterPlaceholders(sql string) string {
	isIdent := func(ch byte) bool {
		return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_'
	}

	var b strings.Builder

	inSingle, inDouble := false, false

	for i := 0; i < len(sql); i++ {
		ch := sql[i]
		if ch == '\'' && !inDouble {
			inSingle = !inSingle

			b.WriteByte(ch)

			continue
		}

		if ch == '"' && !inSingle {
			inDouble = !inDouble

			b.WriteByte(ch)

			continue
		}

		if !inSingle && !inDouble {
			if ch == '?' {
				b.WriteByte(ch)

				if i+1 < len(sql) && isIdent(sql[i+1]) {
					b.WriteByte(' ')
				}

				continue
			}

			if ch == '$' {
				j := i + 1
				for j < len(sql) && sql[j] >= '0' && sql[j] <= '9' {
					j++
				}

				if j > i+1 {
					b.WriteString(sql[i:j])

					if j < len(sql) && isIdent(sql[j]) {
						b.WriteByte(' ')
					}

					i = j - 1

					continue
				}
			}
		}

		b.WriteByte(ch)
	}

	return b.String()
}
