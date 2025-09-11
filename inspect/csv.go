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
		_ = w.Write([]string{"name", "alias", "schema", "source", "joinType"})
	}

	for _, t := range res.Tables {
		_ = w.Write([]string{t.Name, t.Alias, t.Schema, t.Source, t.JoinType})
	}

	w.Flush()

	return buf.Bytes(), w.Error()
}
