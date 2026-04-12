package tools

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	idb "github.com/claytonaalves/firebird-mcp/internal/db"
)

type StatementResult struct {
	Statement    string   `json:"statement"`
	Columns      []string `json:"columns,omitempty"`
	Rows         [][]any  `json:"rows,omitempty"`
	RowsAffected *int64   `json:"rows_affected,omitempty"`
	Error        string   `json:"error,omitempty"`
}

type executeSQLArgs struct {
	SQL string `json:"sql"`
}

func ExecuteSQL(ctx context.Context, db *sql.DB, rawArgs json.RawMessage) (any, error) {
	var args executeSQLArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	if args.SQL == "" {
		return nil, fmt.Errorf("sql is required")
	}

	statements := splitStatements(args.SQL)
	results := make([]StatementResult, 0, len(statements))
	for _, stmt := range statements {
		results = append(results, executeOne(ctx, db, stmt))
	}
	return results, nil
}

func executeOne(ctx context.Context, db *sql.DB, stmt string) StatementResult {
	r := StatementResult{Statement: stmt}
	upper := strings.ToUpper(strings.TrimSpace(stmt))
	if strings.HasPrefix(upper, "SELECT") || strings.HasPrefix(upper, "WITH") {
		cols, rows, err := idb.RunQuery(ctx, db, stmt)
		if err != nil {
			r.Error = err.Error()
			return r
		}
		r.Columns = cols
		if rows == nil {
			rows = [][]any{}
		}
		r.Rows = rows
	} else {
		affected, err := idb.RunExec(ctx, db, stmt)
		if err != nil {
			r.Error = err.Error()
			return r
		}
		r.RowsAffected = &affected
	}
	return r
}

func splitStatements(s string) []string {
	parts := strings.Split(s, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
