package pgxquery

import (
	"context"
	"regexp"

	"github.com/jackc/pgx/v5"
)

// sentinelRe matches a substitution sentinel of the form
// /* ... query.<name> ... */, capturing the prefix, the variable name,
// and the suffix separately so the author's chosen connective /
// separator (e.g. "AND", "OR", ",") is preserved verbatim around the
// substituted fragment. Non-matching comments are left alone.
var sentinelRe = regexp.MustCompile(`/\*\s*([^*]*?)\bquery\.(\w+)\b([^*]*?)\s*\*/`)

var _ pgx.QueryRewriter = &QueryRewriter{}

// QueryRewriter wraps a base list-query params value with AIP additions for
// filter / order / extra positional arguments. It implements
// [pgx.QueryRewriterRewriter] so the generated QueryRewriter* methods can hand it
// straight to [pgx.Conn.QueryRewriter] / [pgx.Pool.QueryRewriter] as the first argument;
// pgx then calls [QueryRewriter.RewriteQueryRewriter] to substitute the sentinel
// comments in the SQL and append Args to the positional parameters.
type QueryRewriter struct {
	// Where is a WHERE-clause fragment (no "WHERE" prefix, no leading or
	// trailing connective — the sentinel in the SQL declares where the
	// connective sits). Placeholders must use absolute $N numbering
	// starting after the base positional args.
	Where string
	// OrderBy is an ORDER BY fragment (no "ORDER BY" prefix, no leading
	// or trailing comma — the sentinel declares the separator position).
	OrderBy string
	// Args holds the values referenced by $N placeholders in Where and
	// OrderBy, in positional order. They are appended to the base args.
	Args []any
}

// RewriteQuery implements [pgx.QueryRewriterRewriter]. For every sentinel of
// the form /* ... query.<name> ... */ found in sql, if the named value
// is non-empty, the sentinel is replaced with its own body (with
// query.<name> swapped for the value) so the SQL author's chosen
// connective / separator is preserved. If the value is empty, the
// sentinel is dropped entirely. Args is appended to the positional args.
func (x *QueryRewriter) RewriteQuery(_ context.Context, _ *pgx.Conn, sql string, args []any) (string, []any, error) {
	sql = sentinelRe.ReplaceAllStringFunc(sql, func(match string) string {
		m := sentinelRe.FindStringSubmatch(match)
		prefix, name, suffix := m[1], m[2], m[3]

		var value string
		switch name {
		case "where":
			value = x.Where
		case "order_by":
			value = x.OrderBy
		}
		if value == "" {
			return ""
		}
		return prefix + value + suffix
	})
	return sql, append(args, x.Args...), nil
}
