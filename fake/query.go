package fake

import (
	"embed"

	"github.com/jackc/pgx/v5"
)

//go:embed *.sql
var fs embed.FS

// NewFakeQuery creates a new fake query.
func NewFakeQuery(name string, args ...any) *pgx.QueuedQuery {
	data, err := fs.ReadFile(name)
	if err != nil {
		panic(err)
	}

	return &pgx.QueuedQuery{
		SQL:       string(data),
		Arguments: args,
	}
}
