/*#
function_name: AccountGet
response_affinity: one
parameters:
  account_id: int
*/
SELECT
    id,
    name,
    status
FROM accounts
WHERE id = /*= account_id */1;
