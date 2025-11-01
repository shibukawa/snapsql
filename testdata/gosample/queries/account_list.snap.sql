/*#
function_name: AccountList
response_affinity: many
*/
SELECT
    id,
    name,
    status
FROM accounts
ORDER BY id DESC;
