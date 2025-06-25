-- SnapSQL拡張を含むテンプレート（2-way SQL形式）
SELECT 
    id,
    name,
    /*# if include_email */
        email,
    /*# end */
    /*# if include_profile */
        profile_image,
        bio
    /*# end */
    /*# for field : additional_fields */
        /*= field */
    /*# end */
FROM users_/*= table_suffix */test
WHERE active = /*= filters.active */true
    AND department IN (/*= filters.departments */'sales', 'marketing')
ORDER BY 
    /*# for sort : sort_fields */
        /*= sort.field */ /*= sort.direction */
    /*# end */name ASC
LIMIT /*= pagination.limit */10
OFFSET /*= pagination.offset */5;
