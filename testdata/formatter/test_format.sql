select u.id,u.name,p.title from users u join posts p on u.id=p.user_id where u.active=true and p.published=true order by p.created_at desc
