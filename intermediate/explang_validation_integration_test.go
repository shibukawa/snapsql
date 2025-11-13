package intermediate

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	snapsql "github.com/shibukawa/snapsql"
)

func TestGenerateFromSQL_ExplangValidationSuccess(t *testing.T) {
	sql := `/*# parameters: { user: { id: int } } */
SELECT id FROM users WHERE user_id = /*= user.id */1`

	cfg := &snapsql.Config{Dialect: "postgres"}

	format, err := GenerateFromSQL(strings.NewReader(sql), nil, "", "", nil, cfg)
	require.NoError(t, err)
	require.NotNil(t, format)
	require.Equal(t, len(format.CELExpressions), len(format.Expressions))

	if len(format.Expressions) > 0 {
		require.NotEmpty(t, format.Expressions[0].Steps)
	}
}
