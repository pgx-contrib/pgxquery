-- sqlfluff:dialect:postgres
-- sqlfluff:max_line_length:1024
-- sqlfluff:rules:capitalisation.keywords:capitalisation_policy:upper

SELECT
    id,
    role,
    company
FROM
    users
WHERE
    id::text >= $1::text
    AND ($2::text IS NULL OR company::text = $2::text)
    /* AND query.where */
ORDER BY
    id
    /* , query.order_by */
LIMIT
    $4::int
    OFFSET
    $3::int;
