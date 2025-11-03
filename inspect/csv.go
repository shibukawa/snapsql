package inspect

import (
	"bytes"
	"encoding/csv"
)

// TablesCSV renders only table list to CSV with a header row.
func TablesCSV(res InspectResult, withHeader bool) ([]byte, error) {
	buf := &bytes.Buffer{}
	w := csv.NewWriter(buf)

	if withHeader {
		_ = w.Write([]string{"name", "alias", "schema", "source", "joinType", "queryName", "isTable"})
	}

	for _, t := range res.Tables {
		isTableStr := "false"
		if t.IsTable {
			isTableStr = "true"
		}

		_ = w.Write([]string{t.Name, t.Alias, t.Schema, t.Source, t.JoinType, t.QueryName, isTableStr})
	}

	w.Flush()

	return buf.Bytes(), w.Error()
}
