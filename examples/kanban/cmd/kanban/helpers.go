package main

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var ErrDSNError = errors.New("invalid dsn")
var ErrInMemoryDSN = errors.New("in-memory sqlite DSN is unsupported for file operations")
var ErrNilFilesystem = errors.New("nil filesystem provided")

func extractSQLiteFilePath(dsn string) (string, error) {
	if dsn == "" {
		return "", fmt.Errorf("%w: empty dsn", ErrDSNError)
	}

	var pathPart string

	hasFilePrefix := false

	if after, ok := strings.CutPrefix(dsn, "file:"); ok {
		pathPart = after

		if strings.HasPrefix(pathPart, "//") {
			pathPart = pathPart[2:]
			if !strings.HasPrefix(pathPart, "/") {
				pathPart = "/" + pathPart
			}
		}
	} else {
		pathPart = dsn
	}

	if strings.HasPrefix(pathPart, ":memory:") {
		return "", ErrInMemoryDSN
	}

	if idx := strings.Index(pathPart, "?"); idx >= 0 {
		pathPart = pathPart[:idx]
	}

	if pathPart == "" {
		return "", fmt.Errorf("cannot derive sqlite path: %w", ErrDSNError)
	}

	if hasFilePrefix {
		return filepath.Clean(filepath.FromSlash(pathPart)), nil
	}

	return filepath.Clean(pathPart), nil
}

func prepareSQLiteFile(dsn string) (string, error) {
	pathPart, err := extractSQLiteFilePath(dsn)
	if err != nil {
		return "", err
	}

	absPath, err := filepath.Abs(pathPart)
	if err != nil {
		return "", err
	}

	dir := filepath.Dir(absPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
	}

	file, err := os.OpenFile(absPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return "", err
	}

	if err := file.Close(); err != nil {
		return "", err
	}

	return absPath, nil
}

func newSPAHandlerFromFS(spaFS fs.FS) (*spaHandler, error) {
	if spaFS == nil {
		return nil, ErrNilFilesystem
	}

	if _, err := fs.Stat(spaFS, "index.html"); err != nil {
		panic("index.html not found in provided filesystem")
	}

	return &spaHandler{
		fs:         spaFS,
		fileServer: http.FileServer(http.FS(spaFS)),
	}, nil
}
