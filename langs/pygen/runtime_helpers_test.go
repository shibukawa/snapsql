package pygen

import "testing"

func renderRuntimeCode(t *testing.T) string {
	t.Helper()

	code, err := RenderRuntimeModule()
	if err != nil {
		t.Fatalf("RenderRuntimeModule() error = %v", err)
	}

	return code
}
