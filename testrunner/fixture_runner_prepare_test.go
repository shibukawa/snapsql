package testrunner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/shibukawa/snapsql/testrunner/fixtureexecutor"
	"github.com/stretchr/testify/require"
)

func TestRunAllFixtureTestsAggregatesPreparationErrors(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()

	const brokenSQL = "# Broken Query\n\n## Description\n\nBroken input used to verify aggregation.\n\n## SQL\n\n```sql\nSELECT id FROM\n```\n\n## Test Cases\n\n### Broken case\n\n**Parameters:**\n```yaml\nid: 1\n```\n\n**Expected Results:**\n```yaml\n- id: 1\n```\n"

	fileNames := []string{"broken_a.snap.md", "broken_b.snap.md"}
	for _, name := range fileNames {
		path := filepath.Join(projectRoot, name)
		require.NoError(t, os.WriteFile(path, []byte(brokenSQL), 0o644))
	}

	runner := NewFixtureTestRunner(projectRoot, nil, "sqlite")
	runner.SetVerbose(false)

	summary, err := runner.RunAllFixtureTests(context.Background())
	require.NoError(t, err)

	require.Equal(t, len(fileNames), summary.TotalTests)
	require.Equal(t, len(fileNames), summary.FailedTests)
	require.Equal(t, len(fileNames), summary.DefinitionFailures)
	require.Len(t, summary.Results, len(fileNames))

	for _, result := range summary.Results {
		require.False(t, result.Success)
		require.Error(t, result.Error)
		require.Equal(t, fixtureexecutor.FailureKindDefinition, result.FailureKind)
	}
}

func TestRunAllFixtureTestsVerboseLists(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()

	const brokenSQL = "# Verbose Query\n\n## Description\n\nListing check for verbose output.\n\n## SQL\n\n```sql\nSELECT id FROM\n```\n\n## Test Cases\n\n### Broken case\n\n**Parameters:**\n```yaml\nid: 1\n```\n\n**Expected Results:**\n```yaml\n- id: 1\n```\n"

	path := filepath.Join(projectRoot, "broken.snap.md")
	require.NoError(t, os.WriteFile(path, []byte(brokenSQL), 0o644))

	runner := NewFixtureTestRunner(projectRoot, nil, "sqlite")
	runner.SetVerbose(true)

	output := captureStdout(t, func() {
		summary, err := runner.RunAllFixtureTests(context.Background())
		require.NoError(t, err)
		require.NotNil(t, summary)
	})

	require.Contains(t, output, "Discovered markdown tests (files: 1, cases: 1)")
	require.Contains(t, output, "  broken.snap.md")
	require.Contains(t, output, "    - Broken case")
}
