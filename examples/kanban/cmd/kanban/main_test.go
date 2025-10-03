package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/shibukawa/snapsql/examples/kanban/internal/handler"
)

func TestExtractSQLiteFilePath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		dsn       string
		wantPath  string
		wantError bool
	}{
		{
			name:     "FileDSNRelative",
			dsn:      "file:kanban.db?cache=shared&mode=rwc",
			wantPath: "kanban.db",
		},
		{
			name:     "PlainRelative",
			dsn:      "./data/kanban.db",
			wantPath: filepath.Clean("./data/kanban.db"),
		},
		{
			name:      "MemoryDSN",
			dsn:       "file::memory:",
			wantError: true,
		},
		{
			name:     "AbsoluteFile",
			dsn:      "file:/tmp/kanban.db",
			wantPath: filepath.Clean("/tmp/kanban.db"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := extractSQLiteFilePath(tc.dsn)
			if tc.wantError {
				if err == nil {
					t.Fatalf("expected error but got nil (path=%s)", got)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if filepath.Clean(got) != tc.wantPath {
				t.Fatalf("path mismatch: got=%q want=%q", got, tc.wantPath)
			}
		})
	}
}

func TestPrepareSQLiteFileCreatesFile(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	target := filepath.Join(tmp, "kanban.db")

	gotPath, err := prepareSQLiteFile(target)
	if err != nil {
		t.Fatalf("prepareSQLiteFile returned error: %v", err)
	}

	if filepath.Clean(gotPath) != filepath.Clean(target) {
		t.Fatalf("unexpected path: got=%q want=%q", gotPath, target)
	}

	if _, err := os.Stat(gotPath); err != nil {
		t.Fatalf("stat after prepare failed: %v", err)
	}

	if err := os.Remove(gotPath); err != nil {
		t.Fatalf("remove after check failed: %v", err)
	}

	dsn := "file:" + filepath.ToSlash(target) + "?cache=shared&mode=rwc"

	gotPath2, err := prepareSQLiteFile(dsn)
	if err != nil {
		t.Fatalf("prepareSQLiteFile(file:...) returned error: %v", err)
	}

	if filepath.Clean(gotPath2) != filepath.Clean(target) {
		t.Fatalf("unexpected path: got=%q want=%q", gotPath2, target)
	}

	if _, err := os.Stat(gotPath2); err != nil {
		t.Fatalf("stat after second prepare failed: %v", err)
	}
}

func TestSPAHandlerServesIndexFallback(t *testing.T) {
	t.Parallel()

	files := fstest.MapFS{
		"index.html":    {Data: []byte("<html><body>spa</body></html>")},
		"assets/app.js": {Data: []byte("console.log('ok')")},
	}

	h, err := newSPAHandlerFromFS(files)
	if err != nil {
		t.Fatalf("newSPAHandlerFromFS error: %v", err)
	}

	t.Run("FallbackToIndex", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected status: %d", rec.Code)
		}

		if !strings.Contains(rec.Body.String(), "spa") {
			t.Fatalf("response should contain index content, got=%q", rec.Body.String())
		}
	})

	t.Run("APINotServed", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, handler.APIPrefix+"/boards", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404 for API prefix, got %d", rec.Code)
		}
	})
}
