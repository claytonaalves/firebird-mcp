package db

import (
	"context"
	"database/sql"
)

// RunQuery and RunExec each run inside their own explicit transaction that is
// committed (or rolled back) before returning. Without this, the underlying
// firebirdsql driver keeps every statement issued on a pooled connection
// inside a single connection-lifetime transaction, "committing" between
// statements only via commit-retaining — which never actually closes the
// transaction and leaves it pinned open (blocking Firebird's garbage
// collection) for as long as the connection stays in the pool. Wrapping each
// call in a real BeginTx/Commit forces a short-lived transaction per call.
func RunQuery(ctx context.Context, db *sql.DB, query string, args ...any) ([]string, [][]any, error) {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}

	cols, err := rows.Columns()
	if err != nil {
		rows.Close()
		return nil, nil, err
	}

	var results [][]any
	for rows.Next() {
		dest := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range dest {
			ptrs[i] = &dest[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			rows.Close()
			return nil, nil, err
		}
		results = append(results, dest)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, nil, err
	}
	rows.Close()

	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	return cols, results, nil
}

func RunExec(ctx context.Context, db *sql.DB, query string, args ...any) (int64, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return affected, nil
}
