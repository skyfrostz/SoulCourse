package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

func (r *ForumRepository) exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return r.db.ExecContext(ctx, rebind(query), args...)
}

func (r *ForumRepository) query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return r.db.QueryContext(ctx, rebind(query), args...)
}

func (r *ForumRepository) queryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return r.db.QueryRowContext(ctx, rebind(query), args...)
}

func execTx(ctx context.Context, tx *sql.Tx, query string, args ...any) (sql.Result, error) {
	return tx.ExecContext(ctx, rebind(query), args...)
}

func queryTx(ctx context.Context, tx *sql.Tx, query string, args ...any) (*sql.Rows, error) {
	return tx.QueryContext(ctx, rebind(query), args...)
}

func queryRowTx(ctx context.Context, tx *sql.Tx, query string, args ...any) *sql.Row {
	return tx.QueryRowContext(ctx, rebind(query), args...)
}

func rebind(query string) string {
	var result strings.Builder
	result.Grow(len(query) + 16)
	parameter := 1
	inSingleQuote := false
	inDoubleQuote := false
	for index := 0; index < len(query); index++ {
		character := query[index]
		if character == '\'' && !inDoubleQuote {
			if inSingleQuote && index+1 < len(query) && query[index+1] == '\'' {
				result.WriteString("''")
				index++
				continue
			}
			inSingleQuote = !inSingleQuote
			result.WriteByte(character)
			continue
		}
		if character == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			result.WriteByte(character)
			continue
		}
		if character == '?' && !inSingleQuote && !inDoubleQuote {
			_, _ = fmt.Fprintf(&result, "$%d", parameter)
			parameter++
			continue
		}
		result.WriteByte(character)
	}
	return result.String()
}
