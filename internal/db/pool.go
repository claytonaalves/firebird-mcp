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

// Registry holds named database pools. Names are ordered as provided at startup.
type Registry struct {
	dbs   map[string]*sql.DB
	names []string
}

// NewRegistry parses args of the form "name=dsn" or bare "dsn" (name → "default").
func NewRegistry(args []string) (*Registry, error) {
	r := &Registry{dbs: make(map[string]*sql.DB)}
	for _, arg := range args {
		name, dsn, found := strings.Cut(arg, "=")
		if !found {
			dsn = name
			name = "default"
		}
		if _, dup := r.dbs[name]; dup {
			return nil, fmt.Errorf("duplicate database name: %s", name)
		}
		pool, err := NewPool(dsn)
		if err != nil {
			r.Close()
			return nil, fmt.Errorf("database %q: %w", name, err)
		}
		r.dbs[name] = pool
		r.names = append(r.names, name)
	}
	if len(r.names) == 0 {
		return nil, fmt.Errorf("at least one database is required")
	}
	return r, nil
}

// Get returns the named pool. If name is empty, the first pool is returned.
func (r *Registry) Get(name string) (*sql.DB, bool) {
	if name == "" {
		return r.dbs[r.names[0]], true
	}
	db, ok := r.dbs[name]
	return db, ok
}

// Names returns the ordered list of database names.
func (r *Registry) Names() []string { return r.names }

// Close closes all pools.
func (r *Registry) Close() {
	for _, db := range r.dbs {
		db.Close()
	}
}
