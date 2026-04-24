package pgxquery_test

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgx-contrib/pgxquery"
)

func ExampleQueryRewriter() {
	type User struct {
		ID   int    `db:"id"`
		Name string `db:"name"`
		Role string `db:"role"`
	}

	config, err := pgxpool.ParseConfig(os.Getenv("PGX_DATABASE_URL"))
	if err != nil {
		panic(err)
	}

	ctx := context.TODO()
	// Create a new pgxpool with the config
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		panic(err)
	}
	// close the pool
	defer pool.Close()

	rows, err := pool.Query(ctx,
		`SELECT id, name, role
		   FROM users
		  WHERE tenant_id = $1
		    /* AND query.where */
		  ORDER BY id
		    /* , query.order_by */`,
		&pgxquery.QueryRewriter{
			Where:   "role = 'admin'",
			OrderBy: "name asc",
			Args:    []any{42},
		},
	)
	if err != nil {
		panic(err)
	}
	// close the rows
	defer rows.Close()

	for rows.Next() {
		entity, err := pgx.RowToStructByName[User](rows)
		if err != nil {
			panic(err)
		}

		fmt.Println(entity.Name)
	}
}
