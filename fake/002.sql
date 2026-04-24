-- sqlfluff:dialect:postgres

SELECT
    *
FROM
    t
WHERE
    a = 1
    /* OR query.where */;
