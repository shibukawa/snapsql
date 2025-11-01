/*#
function_name: UpdateAccountStatusConditional
parameters:
  status: string
  account_id: int
  include_filter: bool
response:
  affinity: none
*/
UPDATE accounts
SET status = /*= status */
WHERE
/*# if include_filter */
    id = /*= account_id */
/*# end */;
