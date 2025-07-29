/*#
function_name: updateUserWithoutLockNo
parameters:
  name: string
  email: string
*/
UPDATE users SET name = /*= name */'John Doe', email = /*= email */'john@example.com' WHERE id = 1
