# pgxquery

[![CI](https://github.com/pgx-contrib/pgxquery/actions/workflows/ci.yml/badge.svg)](https://github.com/pgx-contrib/pgxquery/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/pgx-contrib/pgxquery?include_prereleases)](https://github.com/pgx-contrib/pgxquery/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/pgx-contrib/pgxquery.svg)](https://pkg.go.dev/github.com/pgx-contrib/pgxquery)
[![License](https://img.shields.io/github/license/pgx-contrib/pgxquery)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![pgx](https://img.shields.io/badge/pgx-v5-blue)](https://github.com/jackc/pgx)

A [pgx](https://github.com/jackc/pgx) `QueryRewriter` adapter that substitutes
sentinel comments in your SQL with dynamic `WHERE` and `ORDER BY` fragments at
query time. Write your SQL with `/* AND query.where */` and `/* , query.order_by */`
markers, set the `Where` / `OrderBy` strings on a `QueryRewriter`, and let it
splice them in — preserving your chosen connective or separator verbatim.

## Installation

```bash
go get github.com/pgx-contrib/pgxquery
```

## Usage

Place sentinel comments in your SQL at the positions where dynamic fragments
should appear. Each sentinel has the form `/* <prefix> query.<name> <suffix> */`.
The prefix and suffix (whitespace, connectives like `AND`/`OR`, separators like
`,`) are preserved around the substituted value; when the value is empty the
whole sentinel is dropped.

```go
type User struct {
    ID   int    `db:"id"`
    Name string `db:"name"`
    Role string `db:"role"`
}

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
```

Recognised sentinel names are `where` and `order_by`. `Args` are appended to
the positional parameters passed to `pool.Query` — so the `$1` above is filled
from `Args[0]`. When `Where` or `OrderBy` is empty, its sentinel is stripped
entirely, leaving the surrounding SQL valid.

## Development

### DevContainer

Open in VS Code with the Dev Containers extension. The environment provides Go,
PostgreSQL 18, and Nix automatically.

```
PGX_DATABASE_URL=postgres://vscode@postgres:5432/pgxquery?sslmode=disable
```

### Nix

```bash
nix develop          # enter shell with Go
go tool ginkgo run -r
```

### Run tests

```bash
# Unit tests only (no database required)
go tool ginkgo run -r

# With integration tests
export PGX_DATABASE_URL="postgres://localhost/pgxquery?sslmode=disable"
go tool ginkgo run -r
```

## License

[MIT](LICENSE)
