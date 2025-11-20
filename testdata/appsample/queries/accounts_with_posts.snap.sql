/*#
function_name: ListAccountsWithPosts
statement_type: select
response_affinity: many
*/
SELECT
    a.id,
    a.name,
    a.status,
    p.id AS posts__id,
    p.title AS posts__title,
    p.body AS posts__body,
    p.published_at AS posts__published_at
FROM accounts a
LEFT JOIN posts p ON p.account_id = a.id
ORDER BY a.id, p.id;
