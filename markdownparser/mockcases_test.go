package markdownparser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractMockTestCases(t *testing.T) {
	md := "# Sample Query\n\n" +
		"## Description\n\n" +
		"Simple query for mock extraction.\n\n" +
		"## SQL\n\n" +
		"```sql\n" +
		"SELECT id, name FROM boards WHERE id = {{ args.id }};\n" +
		"```\n\n" +
		"## Test Cases\n\n" +
		"### Returns board by id\n\n" +
		"**Parameters:**\n" +
		"```yaml\n" +
		"id: 42\n" +
		"```\n\n" +
		"**Expected Results:**\n" +
		"```yaml\n" +
		"- id: 42\n" +
		"  name: \"Board 42\"\n" +
		"```\n"

	doc, err := Parse(strings.NewReader(md))
	require.NoError(t, err)

	cases := ExtractMockTestCases(doc)
	require.Len(t, cases, 1)

	mockCase := cases[0]
	require.Equal(t, "Returns board by id", mockCase.Name)
	require.Contains(t, mockCase.Parameters, "id")
	require.EqualValues(t, 42, mockCase.Parameters["id"])
	require.Len(t, mockCase.Responses, 1)
	require.Len(t, mockCase.Responses[0].Expected, 1)
	require.Equal(t, "Board 42", mockCase.Responses[0].Expected[0]["name"])
}
