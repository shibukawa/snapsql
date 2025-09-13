/*#
function_name: my_items_by_id
parameters:
  mid: int
*/
SELECT id FROM items WHERE id = /*= mid */0 ORDER BY id;
