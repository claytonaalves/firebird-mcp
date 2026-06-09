package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/nakagami/firebirdsql"
)

func applyDSNDefaults(dsn string) string {
	atIdx := strings.Index(dsn, "@")
	if atIdx == -1 {
		return "sysdba:masterkey@" + dsn
	}
	userInfo := dsn[:atIdx]
	if !strings.Contains(userInfo, ":") {
		return userInfo + ":masterkey@" + dsn[atIdx+1:]
	}
	return dsn
}

func NewPool(dsn string) (*sql.DB, error) {
	dsn = applyDSNDefaults(dsn)
	db, err := sql.Open("firebirdsql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return db, nil
}
