/*#
function_name: pg_items_by_id
parameters:
  pid: int
*/
SELECT id FROM items WHERE id = /*= pid */0 ORDER BY id;
