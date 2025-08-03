/*#
function_name: get_user_posts
parameters:
  user_id: int
  include_drafts: bool
*/
select u.id,u.name /*# if include_drafts */ ,p.title,p.status /*# else */ ,p.title /*# end */ from users u join posts p on u.id=p.user_id where u.id=/*= user_id */ /*# if !include_drafts */ and p.status='published' /*# end */ order by p.created_at desc
