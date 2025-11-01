/*#
function_name: update_accounts_if_else
parameters:
  status: string
  account_id: int
  enforce_target: bool
*/
UPDATE accounts
SET status = /*= status */'active'
WHERE
/*# if enforce_target */
  id = /*= account_id */1
/*# else */
  1 = 1
/*# end */;
