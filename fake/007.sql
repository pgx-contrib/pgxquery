-- sqlfluff:dialect:postgres

SELECT
    1 /* regular comment */
FROM
    t
WHERE
    TRUE
    /* AND query.where */;
