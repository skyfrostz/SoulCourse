package handler

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

func (h *AdminHandler) exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return h.db.ExecContext(ctx, bindDatabaseQuery(h.cfg.DatabaseDriver, query), args...)
}

func (h *AdminHandler) query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return h.db.QueryContext(ctx, bindDatabaseQuery(h.cfg.DatabaseDriver, query), args...)
}

func (h *AdminHandler) queryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return h.db.QueryRowContext(ctx, bindDatabaseQuery(h.cfg.DatabaseDriver, query), args...)
}

func bindDatabaseQuery(driverName string, query string) string {
	if driverName != "postgres" {
		return query
	}
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
