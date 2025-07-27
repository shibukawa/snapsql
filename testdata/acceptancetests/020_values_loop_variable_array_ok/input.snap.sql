/*#
function_name: insertUserTags
parameters:
  users: ./User[]
*/
INSERT INTO user_tags (user_id, tag) 
VALUES /*# for user : users */(/*= user.id */, /*= user.tags */) /*# end */
