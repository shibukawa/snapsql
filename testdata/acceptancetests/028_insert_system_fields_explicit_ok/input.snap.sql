/*#
function_name: insertUser
parameters:
  user: ./User
  created_at: timestamp
  updated_at: timestamp
*/
INSERT INTO users (id, name, created_at, updated_at) VALUES (/*= user.id */1, /*= user.name */'John', /*= created_at */NOW(), /*= updated_at */NOW())
