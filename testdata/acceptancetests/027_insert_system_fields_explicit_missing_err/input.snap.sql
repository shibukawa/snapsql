/*#
function_name: insertUser
parameters:
  user: ./User
*/
INSERT INTO users (id, name) VALUES (/*= user.id */1, /*= user.name */'John')
