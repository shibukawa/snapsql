/*#
function_name: updateUser
parameters:
  name: string
  email: string
  lock_no: int
*/
UPDATE users SET name = /*= name */'John Doe', email = /*= email */'john@example.com', lock_no = /*= lock_no */2 WHERE id = 1
