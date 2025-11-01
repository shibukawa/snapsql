/*#
function_name: AccountUpdate
response_affinity: many
parameters:
  account_id: int
  status: string
*/
UPDATE accounts
SET status = /*= status */
WHERE id = /*= account_id */
RETURNING id, status;
