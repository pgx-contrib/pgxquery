-- sqlfluff:dialect:postgres

SELECT
    *
FROM
    t
WHERE
    tenant = $1
    /* AND query.where */;
