// Package pgxquery provides a [pgx.QueryRewriter] that substitutes
// sentinel comments in SQL with dynamic WHERE and ORDER BY fragments
// at query time.
//
// Sentinels have the form /* <prefix> query.<name> <suffix> */, where
// the prefix and suffix (whitespace, connectives like AND/OR,
// separators like ",") are preserved verbatim around the substituted
// value. When the value is empty the sentinel is dropped entirely,
// leaving the surrounding SQL valid.
//
// Recognised names are "where" and "order_by", mapped to
// [QueryRewriter.Where] and [QueryRewriter.OrderBy] respectively.
// [QueryRewriter.Args] is appended to the positional arguments passed
// to pgx; any $N placeholders inside Where or OrderBy must use
// absolute numbering starting after the base positional args.
//
// Non-matching SQL comments are left untouched.
//
// Example:
//
//	rows, err := pool.Query(ctx,
//	    `SELECT id, name FROM users
//	       WHERE tenant_id = $1
//	         /* AND query.where */
//	       ORDER BY id
//	         /* , query.order_by */`,
//	    &pgxquery.QueryRewriter{
//	        Where:   "role = 'admin'",
//	        OrderBy: "name asc",
//	        Args:    []any{42},
//	    },
//	)
//
// [pgx.QueryRewriter]: https://pkg.go.dev/github.com/jackc/pgx/v5#QueryRewriter
package pgxquery
