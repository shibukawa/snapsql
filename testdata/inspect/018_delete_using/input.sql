DELETE FROM users u USING logs l WHERE l.user_id = u.id;

