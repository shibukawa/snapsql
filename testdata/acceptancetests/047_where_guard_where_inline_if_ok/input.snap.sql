/*#
function_name: update_accounts_where_inline
parameters:
  status: string
  include_filter: bool
*/
UPDATE accounts
SET status = /*= status */'active'
WHERE /*# if include_filter */ status = /*= status */'active' /*# end */;
