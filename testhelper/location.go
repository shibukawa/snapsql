package testhelper

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
)

func GetCaller(t *testing.T) string {
	t.Helper()

	_, file, line, ok := runtime.Caller(1)
	if !ok {
		return "unknown"
	}

	return fmt.Sprintf("(%s:%d)", filepath.Base(file), line)
}
