/*#
function_name: AccountDelete
statement_type: delete
response_affinity: none
parameters:
  account_id: int
  account_name: string
  include_identifier_filter: bool
  prefer_name_filter: bool
*/
DELETE FROM accounts
WHERE
/*# if include_identifier_filter */
    /*# if prefer_name_filter */
    name = /*= account_name */
    /*# else */
    id = /*= account_id */
    /*# end */
/*# end */;
