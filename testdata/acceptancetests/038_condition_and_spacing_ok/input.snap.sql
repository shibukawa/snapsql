/*#
name: listUserNotifications
function_name: listUserNotifications
parameters:
  user_id: string
  unread_only: bool
  since: string
  has_since: bool
*/
SELECT n.id, n.title
FROM inbox i
WHERE i.user_id = /*= user_id */'EMP001'
/*# if unread_only */
AND i.read_at IS NULL
/*# end */
/*# if has_since */
AND i.created_at > /*= since */'2025-01-01 00:00:00'
/*# end */
AND i.deleted_at IS NULL
