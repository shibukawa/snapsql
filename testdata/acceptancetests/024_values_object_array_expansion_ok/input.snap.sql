/*#
function_name: insertUsers
parameters:
  users: ./User[]
*/
INSERT INTO users (id, name, email) VALUES (/*= users */)
