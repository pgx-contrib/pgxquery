-- sqlfluff:dialect:postgres

SELECT
    *
FROM
    t
WHERE
    id = $1;
