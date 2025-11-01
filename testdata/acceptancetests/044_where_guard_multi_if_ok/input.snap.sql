/*#
function_name: update_accounts_multi_if
parameters:
  status: string
  account_id: int
  include_primary: bool
  include_status: bool
*/
UPDATE accounts
SET status = /*= status */'active'
WHERE 1 = 1
/*# if include_primary */
  AND id = /*= account_id */1
/*# end */
/*# if include_status */
  AND status = /*= status */'active'
/*# end */;
