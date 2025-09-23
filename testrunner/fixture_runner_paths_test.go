package testrunner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindMarkdownTestFilesWithIncludePaths(t *testing.T) {
	projectRoot := t.TempDir()

	dirA := filepath.Join(projectRoot, "a")
	require.NoError(t, os.MkdirAll(dirA, 0o755))
	fileA := filepath.Join(dirA, "alpha_test.snap.md")
	require.NoError(t, os.WriteFile(fileA, []byte("# Test A"), 0o644))

	dirB := filepath.Join(projectRoot, "b")
	require.NoError(t, os.MkdirAll(dirB, 0o755))
	fileB := filepath.Join(dirB, "beta_test.snap.md")
	require.NoError(t, os.WriteFile(fileB, []byte("# Test B"), 0o644))

	runner := &FixtureTestRunner{projectRoot: projectRoot}

	filesAll, err := runner.findMarkdownTestFiles()
	require.NoError(t, err)
	require.ElementsMatch(t, []string{fileA, fileB}, filesAll)

	runner.SetIncludePaths([]string{dirB})
	filesDir, err := runner.findMarkdownTestFiles()
	require.NoError(t, err)
	require.ElementsMatch(t, []string{fileB}, filesDir)

	runner.SetIncludePaths([]string{fileA})
	filesFile, err := runner.findMarkdownTestFiles()
	require.NoError(t, err)
	require.ElementsMatch(t, []string{fileA}, filesFile)
}
