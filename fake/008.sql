-- sqlfluff:dialect:postgres

SELECT
    1
FROM
    t
WHERE
    TRUE
    /* AND query.unknown */;
