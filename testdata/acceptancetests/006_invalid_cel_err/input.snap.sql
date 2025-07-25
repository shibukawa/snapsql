/*#
name: invalidCEL
function_name: invalidCEL
parameters:
  user_id: int
*/
SELECT id, name
FROM users
WHERE id = /*= user_id + (missing_closing_parenthesis */123
