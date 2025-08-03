package intermediate

import (
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/stretchr/testify/require"
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/testhelper"
)

func TestDetermineResponseAffinity(t *testing.T) {
	tests := []struct {
		name             string
		sql              string
		expectedAffinity ResponseAffinity
	}{
		{
			name: "SimpleSelect" + testhelper.GetCaller(t),
			sql: `/*#
name: GetUsers
function_name: getUsers
description: Get all users
*/
SELECT id, name, email FROM users`,
			expectedAffinity: ResponseAffinityMany,
		},
		{
			name: "SelectWithUniqueKey" + testhelper.GetCaller(t),
			sql: `/*#
name: GetUserById
function_name: getUserById
description: Get user by ID
parameters:
  id: int
*/
SELECT id, name, email FROM users WHERE id = 1`,
			expectedAffinity: ResponseAffinityMany, // Will be One when hasUniqueKeyCondition is implemented
		},
		{
			name: "SelectWithLimit1" + testhelper.GetCaller(t),
			sql: `/*#
name: GetFirstUser
function_name: getFirstUser
description: Get first user
*/
SELECT id, name, email FROM users LIMIT 1`,
			expectedAffinity: ResponseAffinityOne,
		},
		{
			name: "InsertWithoutReturning" + testhelper.GetCaller(t),
			sql: `/*#
name: CreateUser
function_name: createUser
description: Create a new user
parameters:
  name: string
  email: string
*/
INSERT INTO users (name, email) VALUES ('John', 'john@example.com')`,
			expectedAffinity: ResponseAffinityNone,
		},
		{
			name: "InsertWithReturning" + testhelper.GetCaller(t),
			sql: `/*#
name: CreateUserReturning
function_name: createUserReturning
description: Create a new user and return ID
parameters:
  name: string
  email: string
*/
INSERT INTO users (name, email) VALUES ('John', 'john@example.com') RETURNING id`,
			expectedAffinity: ResponseAffinityOne,
		},
		{
			name: "BulkInsertWithReturning" + testhelper.GetCaller(t),
			sql: `/*#
name: CreateUsersReturning
function_name: createUsersReturning
description: Create multiple users and return IDs
parameters:
  users: array
*/
INSERT INTO users (name, email) VALUES ('John', 'john@example.com'), ('Jane', 'jane@example.com') RETURNING id`,
			expectedAffinity: ResponseAffinityOne, // Will be Many when isBulkInsert is implemented
		},
		{
			name: "UpdateWithoutReturning" + testhelper.GetCaller(t),
			sql: `/*#
name: UpdateUser
function_name: updateUser
description: Update user name
parameters:
  id: int
  name: string
*/
UPDATE users SET name = 'John' WHERE id = 1`,
			expectedAffinity: ResponseAffinityNone,
		},
		{
			name: "UpdateWithReturning" + testhelper.GetCaller(t),
			sql: `/*#
name: UpdateUserReturning
function_name: updateUserReturning
description: Update user and return data
parameters:
  id: int
  name: string
*/
UPDATE users SET name = 'John' WHERE id = 1 RETURNING id, name`,
			expectedAffinity: ResponseAffinityMany,
		},
		{
			name: "DeleteWithoutReturning" + testhelper.GetCaller(t),
			sql: `/*#
name: DeleteUser
function_name: deleteUser
description: Delete user by ID
parameters:
  id: int
*/
DELETE FROM users WHERE id = 1`,
			expectedAffinity: ResponseAffinityNone,
		},
		{
			name: "DeleteWithReturning" + testhelper.GetCaller(t),
			sql: `/*#
name: DeleteUserReturning
function_name: deleteUserReturning
description: Delete user and return data
parameters:
  id: int
*/
DELETE FROM users WHERE id = 1 RETURNING id, name`,
			expectedAffinity: ResponseAffinityMany,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse SQL using the new ParseSQLFile API
			stmt, _, err := parser.ParseSQLFile(strings.NewReader(tt.sql), nil, ".", ".")
			require.NoError(t, err, "Failed to parse SQL: %s", tt.sql)

			// Determine response affinity
			affinity := determineResponseAffinity(stmt, nil)

			// Verify affinity
			assert.Equal(t, tt.expectedAffinity, affinity, "Response affinity should match")
		})
	}
}
