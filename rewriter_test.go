package pgxquery_test

import (
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

		When("the SQL has no sentinels and Query is zero", func() {
			BeforeEach(func() {
				query = &pgxquery.QueryRewriter{}
			})

			It("returns the sql unchanged and args unchanged", func(ctx SpecContext) {
				sql, args, err := query.RewriteQuery(ctx, nil, "SELECT 1", nil)

				Expect(err).NotTo(HaveOccurred())
				Expect(sql).To(Equal("SELECT 1"))
				Expect(args).To(BeEmpty())
			})
		})

		When("a non-matching comment is present alongside a sentinel", func() {
			It("leaves the regular comment untouched", func(ctx SpecContext) {
				rawSQL := "SELECT 1 /* regular comment */ FROM t /* AND query.where */"
				sql, _, err := query.RewriteQuery(ctx, nil, rawSQL, nil)

				Expect(err).NotTo(HaveOccurred())
				Expect(sql).To(ContainSubstring("/* regular comment */"))
				Expect(sql).To(ContainSubstring("AND role = 'admin'"))
			})
		})

		When("an unknown sentinel name is used", func() {
			It("drops the sentinel entirely", func(ctx SpecContext) {
				rawSQL := "SELECT 1 /* AND query.unknown */ FROM t"
				sql, _, err := query.RewriteQuery(ctx, nil, rawSQL, nil)

				Expect(err).NotTo(HaveOccurred())
				Expect(sql).NotTo(ContainSubstring("query.unknown"))
				Expect(sql).NotTo(ContainSubstring("AND"))
			})
		})

		When("the SQL has no sentinels but Args is set", func() {
			It("appends Args to the positional args", func(ctx SpecContext) {
				sql, args, err := query.RewriteQuery(ctx, nil,
					"SELECT * FROM t WHERE id = $1", []any{1})

				Expect(err).NotTo(HaveOccurred())
				Expect(sql).To(Equal("SELECT * FROM t WHERE id = $1"))
				Expect(args).To(Equal([]any{1, 100, 200}))
			})
		})

		When("the sentinel uses OR as the connective", func() {
			It("preserves OR as the prefix around the substituted value", func(ctx SpecContext) {
				rawSQL := "SELECT * FROM t WHERE a = 1 /* OR query.where */"
				sql, _, err := query.RewriteQuery(ctx, nil, rawSQL, nil)

				Expect(err).NotTo(HaveOccurred())
				Expect(sql).To(ContainSubstring("OR role = 'admin'"))
			})
		})
	})
})
