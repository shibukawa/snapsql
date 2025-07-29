/*#
function_name: getUser
parameters:
  user_id: int
  user: ./User
*/
SELECT u.id, u.name, u.email
FROM users u
WHERE u.id = /*= user_id */1
