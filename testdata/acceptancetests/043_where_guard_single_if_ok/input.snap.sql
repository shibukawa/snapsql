/*#
function_name: update_accounts_single_if
parameters:
  status: string
  account_id: int
  include_filter: bool
*/
UPDATE accounts
SET status = /*= status */'active'
/*# if include_filter */
WHERE id = /*= account_id */1
/*# end */;
