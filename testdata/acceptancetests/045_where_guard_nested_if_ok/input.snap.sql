/*#
function_name: update_accounts_nested_if
parameters:
  status: string
  account_id: int
  include_optional: bool
  include_secondary: bool
*/
UPDATE accounts
SET status = /*= status */'active'
WHERE id = /*= account_id */1
/*# if include_optional */
  AND (
    updated_at > NOW()
    /*# if include_secondary */
      OR status = /*= status */'active'
    /*# end */
  )
/*# end */;
