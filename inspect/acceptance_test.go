package inspect

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestInspectAcceptance(t *testing.T) {
	testRoot := filepath.Join("..", "testdata", "inspect")

	entries, err := os.ReadDir(testRoot)
	if err != nil {
		t.Fatalf("failed to read testdata/inspect: %v", err)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		caseDir := filepath.Join(testRoot, e.Name())
		t.Run(e.Name(), func(t *testing.T) {
			inputPath := filepath.Join(caseDir, "input.sql")

			in, err := os.Open(inputPath)
			if err != nil {
				t.Fatalf("failed to open input: %v", err)
			}
			defer in.Close()

			// Run inspect in strict=true to surface issues; tests should provide valid inputs
			gotRes, err := Inspect(in, InspectOptions{InspectMode: true, Strict: true})
			if err != nil {
				t.Fatalf("inspect error: %v", err)
			}

			// Compare JSON (struct-level)
			expJSONPath := filepath.Join(caseDir, "expected.json")

			expJSON, err := os.ReadFile(expJSONPath)
			if err != nil {
				t.Fatalf("failed to read expected.json: %v", err)
			}

			var expRes InspectResult
			if err := json.Unmarshal(expJSON, &expRes); err != nil {
				t.Fatalf("failed to unmarshal expected.json: %v", err)
			}

			// Write actual for debugging
			actJSONBytes, _ := json.MarshalIndent(gotRes, "", "  ")
			_ = os.WriteFile(filepath.Join(caseDir, "actual.json"), actJSONBytes, 0644)

			assert.Equal(t, expRes.Statement, gotRes.Statement)
			assert.Equal(t, expRes.Tables, gotRes.Tables)
			assert.Equal(t, expRes.Notes, gotRes.Notes)

			// Compare CSV (row-level)
			expCSVPath := filepath.Join(caseDir, "expected.csv")

			expCSVBytes, err := os.ReadFile(expCSVPath)
			if err != nil {
				t.Fatalf("failed to read expected.csv: %v", err)
			}

			expRows, err := csv.NewReader(bytesReader(expCSVBytes)).ReadAll()
			if err != nil {
				t.Fatalf("failed to parse expected.csv: %v", err)
			}

			gotCSV, err := TablesCSV(gotRes, true)
			if err != nil {
				t.Fatalf("failed to build csv: %v", err)
			}

			_ = os.WriteFile(filepath.Join(caseDir, "actual.csv"), gotCSV, 0644)

			gotRows, err := csv.NewReader(bytesReader(gotCSV)).ReadAll()
			if err != nil {
				t.Fatalf("failed to parse actual csv: %v", err)
			}

			assert.Equal(t, expRows, gotRows)
		})
	}
}

// bytesReader wraps a byte slice to an io.Reader without importing bytes in every compare path.
func bytesReader(b []byte) *byteReader { return &byteReader{b: b} }

type byteReader struct {
	b []byte
	i int
}

func (r *byteReader) Read(p []byte) (int, error) {
	if r.i >= len(r.b) {
		return 0, io.EOF
	}

	n := copy(p, r.b[r.i:])
	r.i += n

	return n, nil
}
