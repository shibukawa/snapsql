/*#
function_name: postpone_cards
*/
WITH pending AS (SELECT id FROM cards WHERE status = 'pending')
SELECT id FROM pending
