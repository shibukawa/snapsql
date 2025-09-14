/*#
function_name: insert_user_with_returning_sqlite
parameters:
  user_name: string
  user_email: string
*/
INSERT INTO users (name, email, created_at) 
VALUES (/*= user_name */'John Doe', /*= user_email */'john@example.com', NOW())
RETURNING id, name, email, created_at;
