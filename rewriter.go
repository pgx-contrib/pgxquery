package pgxquery

import (
	"context"
	"regexp"
	"strconv"
	"strings"

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
	// connective sits). Placeholders use local $N numbering: $1 refers
	// to Args[0], $2 to Args[1], and so on. The rewriter shifts them by
	// the count of base positional args so they land in the right slots
	// of the final flat args slice.
	Where string
	// OrderBy is an ORDER BY fragment (no "ORDER BY" prefix, no leading
	// or trailing comma — the sentinel declares the separator position).
	// Placeholders follow the same local $N convention as Where.
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
//
// $N placeholders in Where and OrderBy are rewritten to absolute
// numbering by adding len(args) to N, so fragments can be authored with
// local $1, $2, ... numbering independent of the base query's args.
func (x *QueryRewriter) RewriteQuery(_ context.Context, _ *pgx.Conn, sql string, args []any) (string, []any, error) {
	offset := len(args)
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
		return prefix + shiftPlaceholders(value, offset) + suffix
	})
	return sql, append(args, x.Args...), nil
}

// shiftPlaceholders rewrites every active $N placeholder in s to $(N+offset).
// Placeholders inside string literals ('...'), quoted identifiers ("..."),
// dollar-quoted strings ($tag$...$tag$), line comments (-- ...) and block
// comments (/* ... */) are left untouched, so the scanner is safe against
// false positives like $1 inside 'a$1b' or $body$ ... $1 ... $body$.
func shiftPlaceholders(s string, offset int) string {
	if offset == 0 {
		return s
	}

	var b strings.Builder
	b.Grow(len(s))

	i := 0
	for i < len(s) {
		c := s[i]
		switch {
		case c == '\'':
			j := skipSingleQuoted(s, i)
			b.WriteString(s[i:j])
			i = j
		case c == '"':
			j := skipDoubleQuoted(s, i)
			b.WriteString(s[i:j])
			i = j
		case c == '-' && i+1 < len(s) && s[i+1] == '-':
			j := skipLineComment(s, i)
			b.WriteString(s[i:j])
			i = j
		case c == '/' && i+1 < len(s) && s[i+1] == '*':
			j := skipBlockComment(s, i)
			b.WriteString(s[i:j])
			i = j
		case c == '$':
			if i+1 < len(s) && isDigit(s[i+1]) {
				j := i + 1
				for j < len(s) && isDigit(s[j]) {
					j++
				}
				n, _ := strconv.Atoi(s[i+1 : j])
				b.WriteByte('$')
				b.WriteString(strconv.Itoa(n + offset))
				i = j
				continue
			}
			if j, ok := skipDollarQuoted(s, i); ok {
				b.WriteString(s[i:j])
				i = j
				continue
			}
			b.WriteByte(c)
			i++
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String()
}

func skipSingleQuoted(s string, i int) int {
	j := i + 1
	for j < len(s) {
		if s[j] == '\'' {
			if j+1 < len(s) && s[j+1] == '\'' {
				j += 2
				continue
			}
			return j + 1
		}
		j++
	}
	return j
}

func skipDoubleQuoted(s string, i int) int {
	j := i + 1
	for j < len(s) {
		if s[j] == '"' {
			if j+1 < len(s) && s[j+1] == '"' {
				j += 2
				continue
			}
			return j + 1
		}
		j++
	}
	return j
}

func skipLineComment(s string, i int) int {
	j := i + 2
	for j < len(s) && s[j] != '\n' {
		j++
	}
	if j < len(s) {
		j++
	}
	return j
}

// skipBlockComment handles /* ... */ with PostgreSQL's nesting semantics.
func skipBlockComment(s string, i int) int {
	j := i + 2
	depth := 1
	for j < len(s) && depth > 0 {
		switch {
		case j+1 < len(s) && s[j] == '/' && s[j+1] == '*':
			depth++
			j += 2
		case j+1 < len(s) && s[j] == '*' && s[j+1] == '/':
			depth--
			j += 2
		default:
			j++
		}
	}
	return j
}

// skipDollarQuoted parses a dollar-quoted string starting at i (where s[i]
// is known to be '$'). On success it returns the index just past the
// closing tag and true; otherwise it returns (i, false) so the caller can
// treat the '$' as a literal character.
func skipDollarQuoted(s string, i int) (int, bool) {
	j := i + 1
	for j < len(s) {
		c := s[j]
		if c == '$' {
			break
		}
		if isAlpha(c) || c == '_' || (j > i+1 && isDigit(c)) {
			j++
			continue
		}
		return i, false
	}
	if j >= len(s) || s[j] != '$' {
		return i, false
	}
	tag := s[i : j+1]
	k := j + 1
	idx := strings.Index(s[k:], tag)
	if idx < 0 {
		return len(s), true
	}
	return k + idx + len(tag), true
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }
func isAlpha(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }
