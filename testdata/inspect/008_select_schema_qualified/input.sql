SELECT u.id
FROM public.users u
JOIN sales.orders o ON o.user_id = u.id;

