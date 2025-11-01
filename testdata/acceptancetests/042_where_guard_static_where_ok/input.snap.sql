/*#
function_name: update_accounts_static_where
parameters:
  status: string
  account_id: int
*/
UPDATE accounts
SET status = /*= status */'active'
WHERE id = /*= account_id */1;
