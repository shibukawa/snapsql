SELECT 
    sq.id,
    sq.name
FROM (
    SELECT 
        id,
        name
    FROM users
) AS sq
