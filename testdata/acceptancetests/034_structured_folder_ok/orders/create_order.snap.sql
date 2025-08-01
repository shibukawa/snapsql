/*#
function_name: create_order
description: Create a new order
parameters:
  user_id: int
  total: decimal
  status: string
response_affinity: none
*/
INSERT INTO orders (user_id, total, status, created_at)
VALUES (/*= user_id */1, /*= total */100.50, /*= status */'pending', NOW());
