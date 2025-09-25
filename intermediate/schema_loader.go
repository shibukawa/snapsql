package intermediate

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/shibukawa/snapsql"
)

// LoadSchemaFromSQL parses a lightweight SQLite-style schema.sql and returns TableInfo map.
// Supported subset:
//
//	CREATE TABLE [IF NOT EXISTS] <name> (\n column defs..., [table constraints...] );
//
// Column line pattern examples:
//
//	id INTEGER PRIMARY KEY AUTOINCREMENT,
//	board_id INTEGER NOT NULL,
//	position REAL NOT NULL,
//	name TEXT,
//
// Table level PK pattern: PRIMARY KEY (col1, col2)
// Unknown tokens after type are ignored.
func LoadSchemaFromSQL(path string) (map[string]*snapsql.TableInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Regexes
	createRe := regexp.MustCompile(`(?i)^CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z0-9_]+)\s*\(`)
	tablePKRe := regexp.MustCompile(`(?i)^PRIMARY\s+KEY\s*\(([^)]+)\)`) // captures col list
	// column start: name type ... (PRIMARY KEY optional inline)
	colRe := regexp.MustCompile(`^([a-zA-Z0-9_]+)\s+([a-zA-Z0-9()]+)(.*)$`)

	tables := map[string]*snapsql.TableInfo{}

	var (
		current    *snapsql.TableInfo
		collecting bool
	)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "--") { // skip comments/blank
			continue
		}
		// Normalize line (remove trailing comma for parsing decisions; store original order separately)
		hasComma := strings.HasSuffix(line, ",")
		if hasComma {
			line = strings.TrimSuffix(line, ",")
			line = strings.TrimSpace(line)
		}

		if !collecting {
			if m := createRe.FindStringSubmatch(line); m != nil {
				name := m[1]
				current = &snapsql.TableInfo{Name: name, Columns: map[string]*snapsql.ColumnInfo{}}
				tables[name] = current
				collecting = true
			}

			continue
		}

		// End of CREATE TABLE block
		if strings.HasPrefix(line, ")") { // block ends
			collecting = false
			current = nil

			continue
		}

		if current == nil { // safety
			continue
		}

		// Table-level PRIMARY KEY
		if m := tablePKRe.FindStringSubmatch(strings.ToUpper(line)); m != nil {
			cols := strings.Split(m[1], ",")
			for _, c := range cols {
				colName := strings.TrimSpace(strings.ToLower(c))
				if ci, ok := current.Columns[colName]; ok {
					ci.IsPrimaryKey = true
				}
			}

			continue
		}

		// Column definition
		if m := colRe.FindStringSubmatch(line); m != nil {
			colName := strings.ToLower(m[1])
			rawType := strings.ToUpper(m[2])
			rest := strings.ToUpper(m[3])
			// Map type
			dataType := mapSQLType(rawType)

			colInfo := &snapsql.ColumnInfo{Name: colName, DataType: dataType}
			if strings.Contains(rest, "NOT NULL") {
				colInfo.Nullable = false
			} else {
				colInfo.Nullable = true // default
			}

			if strings.Contains(rest, "PRIMARY KEY") {
				colInfo.IsPrimaryKey = true
			}

			current.Columns[colName] = colInfo
			current.ColumnOrder = append(current.ColumnOrder, colName)

			continue
		}
	}

	return tables, scanner.Err()
}

func mapSQLType(sqlType string) string {
	// strip size e.g. VARCHAR(255)
	if idx := strings.Index(sqlType, "("); idx >= 0 {
		sqlType = sqlType[:idx]
	}

	switch sqlType {
	case "INTEGER", "INT", "BIGINT", "SMALLINT":
		return "int"
	case "REAL", "DOUBLE", "FLOAT", "NUMERIC", "DECIMAL":
		return "float64"
	case "TEXT", "VARCHAR", "CHAR", "CLOB":
		return "string"
	case "DATETIME", "TIMESTAMP":
		return "string"
	default:
		return "any"
	}
}

// discoverSchemaFile attempts to find a schema.sql relative to queries path.
func discoverSchemaFile(queriesInputPath string) string {
	// Candidate 1: <queries>/../sql/schema.sql
	dir := filepath.Dir(queriesInputPath)

	candidate1 := filepath.Join(dir, "sql", "schema.sql")
	if _, err := os.Stat(candidate1); err == nil {
		return candidate1
	}
	// Candidate 2: project root sql/schema.sql (walk up until we find queries parent marker?)
	// Simplified: ascend two levels max
	parent := filepath.Dir(dir)

	candidate2 := filepath.Join(parent, "sql", "schema.sql")
	if _, err := os.Stat(candidate2); err == nil {
		return candidate2
	}

	return ""
}
