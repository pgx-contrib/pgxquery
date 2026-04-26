package pgxquery_test

import (
	"strings"

	"github.com/pgx-contrib/pgxquery"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/pgx-contrib/pgxquery/fake"
)

var _ = Describe("QueryRewriter", func() {
	var query *pgxquery.QueryRewriter

	BeforeEach(func() {
		query = &pgxquery.QueryRewriter{
			Where:   "role = 'admin'",
			OrderBy: "name asc",
			Args:    []any{100, 200},
		}
	})

	Describe("RewriteQuery", func() {
		It("substitutes where and order_by sentinels", func(ctx SpecContext) {
			fq := NewFakeQuery("001.sql", "007", "Google", 0, 10)
			sql, args, err := query.RewriteQuery(ctx, nil, fq.SQL, fq.Arguments)

			Expect(err).NotTo(HaveOccurred())
			Expect(sql).To(ContainSubstring("AND role = 'admin'"))
			Expect(sql).To(ContainSubstring(", name asc"))
			Expect(sql).NotTo(ContainSubstring("query.where"))
			Expect(sql).NotTo(ContainSubstring("query.order_by"))
			Expect(args).To(HaveLen(6))
			Expect(args[4]).To(Equal(100))
			Expect(args[5]).To(Equal(200))
		})

		When("Where is empty", func() {
			BeforeEach(func() {
				query.Where = ""
			})

			It("drops the where sentinel and keeps order_by", func(ctx SpecContext) {
				fq := NewFakeQuery("001.sql", "007", "Google", 0, 10)
				sql, args, err := query.RewriteQuery(ctx, nil, fq.SQL, fq.Arguments)

				Expect(err).NotTo(HaveOccurred())
				Expect(sql).NotTo(ContainSubstring("query.where"))
				Expect(sql).NotTo(ContainSubstring("role = 'admin'"))
				Expect(sql).To(ContainSubstring(", name asc"))
				Expect(args).To(HaveLen(6))
			})
		})

		When("OrderBy is empty", func() {
			BeforeEach(func() {
				query.OrderBy = ""
			})

			It("drops the order_by sentinel and keeps where", func(ctx SpecContext) {
				fq := NewFakeQuery("001.sql", "007", "Google", 0, 10)
				sql, args, err := query.RewriteQuery(ctx, nil, fq.SQL, fq.Arguments)

				Expect(err).NotTo(HaveOccurred())
				Expect(sql).To(ContainSubstring("AND role = 'admin'"))
				Expect(sql).NotTo(ContainSubstring("query.order_by"))
				Expect(sql).NotTo(ContainSubstring("name asc"))
				Expect(args).To(HaveLen(6))
			})
		})

		When("both Where and OrderBy are empty", func() {
			BeforeEach(func() {
				query.Where = ""
				query.OrderBy = ""
			})

			It("drops both sentinels but still appends Args", func(ctx SpecContext) {
				fq := NewFakeQuery("001.sql", "007", "Google", 0, 10)
				sql, args, err := query.RewriteQuery(ctx, nil, fq.SQL, fq.Arguments)

				Expect(err).NotTo(HaveOccurred())
				Expect(sql).NotTo(ContainSubstring("query."))
				Expect(args).To(HaveLen(6))
			})
		})

		When("the sentinel uses OR as the connective", func() {
			It("preserves OR as the prefix around the substituted value", func(ctx SpecContext) {
				fq := NewFakeQuery("002.sql")
				sql, _, err := query.RewriteQuery(ctx, nil, fq.SQL, fq.Arguments)

				Expect(err).NotTo(HaveOccurred())
				Expect(sql).To(ContainSubstring("OR role = 'admin'"))
				Expect(sql).NotTo(ContainSubstring("query.where"))
			})
		})

		When("the sentinel has no prefix or suffix", func() {
			It("substitutes the value alone", func(ctx SpecContext) {
				fq := NewFakeQuery("003.sql")
				sql, _, err := query.RewriteQuery(ctx, nil, fq.SQL, fq.Arguments)

				Expect(err).NotTo(HaveOccurred())
				Expect(sql).To(ContainSubstring("role = 'admin'"))
				Expect(sql).NotTo(ContainSubstring("query.where"))
			})
		})

		When("the order_by sentinel is placed before the static list", func() {
			BeforeEach(func() {
				query.OrderBy = "priority desc"
			})

			It("preserves the trailing comma as the suffix", func(ctx SpecContext) {
				fq := NewFakeQuery("004.sql")
				sql, _, err := query.RewriteQuery(ctx, nil, fq.SQL, fq.Arguments)

				Expect(err).NotTo(HaveOccurred())
				Expect(sql).To(ContainSubstring("priority desc , id"))
			})

			When("OrderBy is empty", func() {
				BeforeEach(func() {
					query.OrderBy = ""
				})

				It("drops the sentinel leaving the static list intact", func(ctx SpecContext) {
					fq := NewFakeQuery("004.sql")
					sql, _, err := query.RewriteQuery(ctx, nil, fq.SQL, fq.Arguments)

					Expect(err).NotTo(HaveOccurred())
					Expect(sql).NotTo(ContainSubstring("query.order_by"))
					Expect(sql).NotTo(ContainSubstring(","))
					Expect(sql).To(ContainSubstring("id"))
				})
			})
		})

		When("the SQL contains multiple sentinels of the same kind", func() {
			It("substitutes each occurrence independently", func(ctx SpecContext) {
				fq := NewFakeQuery("005.sql")
				sql, _, err := query.RewriteQuery(ctx, nil, fq.SQL, fq.Arguments)

				Expect(err).NotTo(HaveOccurred())
				Expect(strings.Count(sql, "AND role = 'admin'")).To(Equal(2))
				Expect(sql).NotTo(ContainSubstring("query.where"))
			})
		})

		When("Where uses local placeholders and Args supplies values", func() {
			BeforeEach(func() {
				query = &pgxquery.QueryRewriter{
					Where: "name = $1 AND score > $2",
					Args:  []any{"alice", 90},
				}
			})

			It("shifts placeholders past base args and appends Args", func(ctx SpecContext) {
				fq := NewFakeQuery("006.sql", "acme")
				sql, args, err := query.RewriteQuery(ctx, nil, fq.SQL, fq.Arguments)

				Expect(err).NotTo(HaveOccurred())
				Expect(sql).To(ContainSubstring("AND name = $2 AND score > $3"))
				Expect(args).To(Equal([]any{"acme", "alice", 90}))
			})
		})

		When("Where contains $N inside string literals and dollar-quoted bodies", func() {
			BeforeEach(func() {
				query = &pgxquery.QueryRewriter{
					Where: "name = $1 AND note = 'literal $1 stays' AND body = $body$raw $1 stays$body$ AND score > $2",
					Args:  []any{"alice", 90},
				}
			})

			It("only shifts active placeholders, leaving quoted bodies untouched", func(ctx SpecContext) {
				fq := NewFakeQuery("006.sql", "acme")
				sql, _, err := query.RewriteQuery(ctx, nil, fq.SQL, fq.Arguments)

				Expect(err).NotTo(HaveOccurred())
				Expect(sql).To(ContainSubstring("name = $2"))
				Expect(sql).To(ContainSubstring("'literal $1 stays'"))
				Expect(sql).To(ContainSubstring("$body$raw $1 stays$body$"))
				Expect(sql).To(ContainSubstring("score > $3"))
			})
		})

		When("Where contains multi-digit placeholders", func() {
			BeforeEach(func() {
				args := make([]any, 12)
				query = &pgxquery.QueryRewriter{
					Where: "x = $1 AND y = $10 AND z = $12",
					Args:  args,
				}
			})

			It("shifts every placeholder by len(base args)", func(ctx SpecContext) {
				fq := NewFakeQuery("006.sql", "acme")
				sql, _, err := query.RewriteQuery(ctx, nil, fq.SQL, fq.Arguments)

				Expect(err).NotTo(HaveOccurred())
				Expect(sql).To(ContainSubstring("x = $2 AND y = $11 AND z = $13"))
			})
		})

		When("OrderBy contains a placeholder", func() {
			BeforeEach(func() {
				query = &pgxquery.QueryRewriter{
					OrderBy: "CASE WHEN role = $1 THEN 0 ELSE 1 END",
					Args:    []any{"admin"},
				}
			})

			It("shifts the placeholder past the base args", func(ctx SpecContext) {
				fq := NewFakeQuery("001.sql", "007", "Google", 0, 10)
				sql, args, err := query.RewriteQuery(ctx, nil, fq.SQL, fq.Arguments)

				Expect(err).NotTo(HaveOccurred())
				Expect(sql).To(ContainSubstring(", CASE WHEN role = $5 THEN 0 ELSE 1 END"))
				Expect(args).To(Equal([]any{"007", "Google", 0, 10, "admin"}))
			})
		})

		When("Where contains $N inside a quoted identifier", func() {
			BeforeEach(func() {
				query = &pgxquery.QueryRewriter{
					Where: `"col$1" = $1`,
					Args:  []any{"x"},
				}
			})

			It("leaves the identifier body untouched and shifts the active placeholder", func(ctx SpecContext) {
				fq := NewFakeQuery("006.sql", "acme")
				sql, _, err := query.RewriteQuery(ctx, nil, fq.SQL, fq.Arguments)

				Expect(err).NotTo(HaveOccurred())
				Expect(sql).To(ContainSubstring(`"col$1" = $2`))
			})
		})

		When("Where contains $N inside line and block comments", func() {
			BeforeEach(func() {
				query = &pgxquery.QueryRewriter{
					Where: "x = $1 -- ignore $1 here\n   AND y = $2 /* and /* nested $1 */ stays */ AND z = $3",
					Args:  []any{1, 2, 3},
				}
			})

			It("only shifts placeholders outside comments", func(ctx SpecContext) {
				fq := NewFakeQuery("006.sql", "acme")
				sql, _, err := query.RewriteQuery(ctx, nil, fq.SQL, fq.Arguments)

				Expect(err).NotTo(HaveOccurred())
				Expect(sql).To(ContainSubstring("x = $2"))
				Expect(sql).To(ContainSubstring("-- ignore $1 here"))
				Expect(sql).To(ContainSubstring("AND y = $3"))
				Expect(sql).To(ContainSubstring("/* and /* nested $1 */ stays */"))
				Expect(sql).To(ContainSubstring("AND z = $4"))
			})
		})

		When("Where contains escaped quotes and a lone dollar sign", func() {
			BeforeEach(func() {
				query = &pgxquery.QueryRewriter{
					Where: `note = 'it''s $1 inside' AND price > $1 AND tag <> 'A' || '$' || 'B'`,
					Args:  []any{42},
				}
			})

			It("treats '' as an escape, leaves the lone $ alone, and shifts active placeholders", func(ctx SpecContext) {
				fq := NewFakeQuery("006.sql", "acme")
				sql, _, err := query.RewriteQuery(ctx, nil, fq.SQL, fq.Arguments)

				Expect(err).NotTo(HaveOccurred())
				Expect(sql).To(ContainSubstring(`'it''s $1 inside'`))
				Expect(sql).To(ContainSubstring("price > $2"))
				Expect(sql).To(ContainSubstring(`'A' || '$' || 'B'`))
			})
		})

		When("Where contains an empty-tag dollar-quoted body", func() {
			BeforeEach(func() {
				query = &pgxquery.QueryRewriter{
					Where: "raw = $$keep $1 raw$$ AND id = $1",
					Args:  []any{7},
				}
			})

			It("leaves $$...$$ bodies untouched and shifts active placeholders", func(ctx SpecContext) {
				fq := NewFakeQuery("006.sql", "acme")
				sql, _, err := query.RewriteQuery(ctx, nil, fq.SQL, fq.Arguments)

				Expect(err).NotTo(HaveOccurred())
				Expect(sql).To(ContainSubstring("$$keep $1 raw$$"))
				Expect(sql).To(ContainSubstring("AND id = $2"))
			})
		})

		When("Where ends with an unterminated string literal", func() {
			BeforeEach(func() {
				query = &pgxquery.QueryRewriter{
					Where: "x = $1 AND note = 'unterminated $1",
					Args:  []any{1},
				}
			})

			It("shifts placeholders before the open quote and leaves the rest opaque", func(ctx SpecContext) {
				fq := NewFakeQuery("006.sql", "acme")
				sql, _, err := query.RewriteQuery(ctx, nil, fq.SQL, fq.Arguments)

				Expect(err).NotTo(HaveOccurred())
				Expect(sql).To(ContainSubstring("x = $2"))
				Expect(sql).To(ContainSubstring("'unterminated $1"))
			})
		})

		When("Where ends with an unterminated dollar-quoted body", func() {
			BeforeEach(func() {
				query = &pgxquery.QueryRewriter{
					Where: "x = $1 AND raw = $tag$body $1 still raw",
					Args:  []any{1},
				}
			})

			It("shifts placeholders before the opener and leaves the rest opaque", func(ctx SpecContext) {
				fq := NewFakeQuery("006.sql", "acme")
				sql, _, err := query.RewriteQuery(ctx, nil, fq.SQL, fq.Arguments)

				Expect(err).NotTo(HaveOccurred())
				Expect(sql).To(ContainSubstring("x = $2"))
				Expect(sql).To(ContainSubstring("$tag$body $1 still raw"))
			})
		})

		When("Where contains lone dollar signs and an unclosed tag at EOF", func() {
			BeforeEach(func() {
				query = &pgxquery.QueryRewriter{
					Where: "x = $1 AND amount > $ AND label = $abc",
					Args:  []any{1},
				}
			})

			It("passes lone $ through and treats an unclosed tag opener as literal", func(ctx SpecContext) {
				fq := NewFakeQuery("006.sql", "acme")
				sql, _, err := query.RewriteQuery(ctx, nil, fq.SQL, fq.Arguments)

				Expect(err).NotTo(HaveOccurred())
				Expect(sql).To(ContainSubstring("x = $2"))
				Expect(sql).To(ContainSubstring("amount > $ "))
				Expect(sql).To(ContainSubstring("label = $abc"))
			})
		})

		When("Where contains a quoted identifier with escaped quotes and an unterminated identifier", func() {
			BeforeEach(func() {
				query = &pgxquery.QueryRewriter{
					Where: `"weird""col $1" = $1 AND "open $1`,
					Args:  []any{1},
				}
			})

			It("treats \"\" as an escape and leaves the unterminated identifier opaque", func(ctx SpecContext) {
				fq := NewFakeQuery("006.sql", "acme")
				sql, _, err := query.RewriteQuery(ctx, nil, fq.SQL, fq.Arguments)

				Expect(err).NotTo(HaveOccurred())
				Expect(sql).To(ContainSubstring(`"weird""col $1" = $2`))
				Expect(sql).To(ContainSubstring(`"open $1`))
			})
		})

		When("the base query has no positional args (offset is zero)", func() {
			BeforeEach(func() {
				query = &pgxquery.QueryRewriter{
					Where: "name = $1 AND score > $2",
					Args:  []any{"alice", 90},
				}
			})

			It("leaves placeholders unchanged", func(ctx SpecContext) {
				sql := `SELECT * FROM t WHERE 1=1 /* AND query.where */`
				out, args, err := query.RewriteQuery(ctx, nil, sql, nil)

				Expect(err).NotTo(HaveOccurred())
				Expect(out).To(ContainSubstring("AND name = $1 AND score > $2"))
				Expect(args).To(Equal([]any{"alice", 90}))
			})
		})

		When("a non-matching comment is present alongside a sentinel", func() {
			It("leaves the regular comment untouched", func(ctx SpecContext) {
				fq := NewFakeQuery("007.sql")
				sql, _, err := query.RewriteQuery(ctx, nil, fq.SQL, fq.Arguments)

				Expect(err).NotTo(HaveOccurred())
				Expect(sql).To(ContainSubstring("/* regular comment */"))
				Expect(sql).To(ContainSubstring("AND role = 'admin'"))
			})
		})

		When("an unknown sentinel name is used", func() {
			It("drops the sentinel entirely", func(ctx SpecContext) {
				fq := NewFakeQuery("008.sql")
				sql, _, err := query.RewriteQuery(ctx, nil, fq.SQL, fq.Arguments)

				Expect(err).NotTo(HaveOccurred())
				Expect(sql).NotTo(ContainSubstring("query.unknown"))
				Expect(sql).NotTo(ContainSubstring("AND"))
			})
		})

		When("the SQL has no sentinels but Args is set", func() {
			It("appends Args to the positional args", func(ctx SpecContext) {
				fq := NewFakeQuery("009.sql", 1)
				sql, args, err := query.RewriteQuery(ctx, nil, fq.SQL, fq.Arguments)

				Expect(err).NotTo(HaveOccurred())
				Expect(sql).To(Equal(fq.SQL))
				Expect(args).To(Equal([]any{1, 100, 200}))
			})
		})

		When("using the 001.sql fixture with both sentinels and extra args", func() {
			It("produces the expected fully rewritten SQL", func(ctx SpecContext) {
				fq := NewFakeQuery("001.sql", "007", "Google", 0, 10)
				sql, args, err := query.RewriteQuery(ctx, nil, fq.SQL, fq.Arguments)

				Expect(err).NotTo(HaveOccurred())
				Expect(sql).To(Equal(NewFakeQuery("001.expected.sql").SQL))
				Expect(args).To(Equal([]any{"007", "Google", 0, 10, 100, 200}))
			})
		})
	})
})
