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

		When("Where uses positional placeholders and Args supplies values", func() {
			BeforeEach(func() {
				query = &pgxquery.QueryRewriter{
					Where: "name = $2 AND score > $3",
					Args:  []any{"alice", 90},
				}
			})

			It("passes placeholders through and appends Args after base args", func(ctx SpecContext) {
				fq := NewFakeQuery("006.sql", "acme")
				sql, args, err := query.RewriteQuery(ctx, nil, fq.SQL, fq.Arguments)

				Expect(err).NotTo(HaveOccurred())
				Expect(sql).To(ContainSubstring("AND name = $2 AND score > $3"))
				Expect(args).To(Equal([]any{"acme", "alice", 90}))
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
				Expect(sql).To(Equal(`-- sqlfluff:dialect:postgres
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
    AND role = 'admin'
ORDER BY
    id
    , name asc
LIMIT
    $4::int
    OFFSET
    $3::int;
`))
				Expect(args).To(Equal([]any{"007", "Google", 0, 10, 100, 200}))
			})
		})
	})
})
