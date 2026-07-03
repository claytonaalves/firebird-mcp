package db

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
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

// sanitizeLabel strips embedded credentials from a DSN for use as a
// registry key / display name, using the same "@" split point that
// applyDSNDefaults uses to separate userinfo from host/path.
func sanitizeLabel(dsn string) string {
	if idx := strings.Index(dsn, "@"); idx != -1 {
		return dsn[idx+1:]
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

// Registry is a cache of database connections opened on demand, keyed by
// sanitized DSN label. Nothing is pre-connected: every entry is created the
// first time it's resolved.
type Registry struct {
	mu    sync.RWMutex
	dbs   map[string]*sql.DB
	names []string
}

func NewRegistry() *Registry {
	return &Registry{dbs: make(map[string]*sql.DB)}
}

// Resolve returns the pool for dsn, opening and caching a new connection on
// first use. Subsequent calls with a DSN that sanitizes to the same label
// reuse the cached pool.
func (r *Registry) Resolve(dsn string) (*sql.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("database is required: pass a DSN/path, e.g. host/path/to/database.fdb")
	}
	label := sanitizeLabel(dsn)

	r.mu.RLock()
	if pool, ok := r.dbs[label]; ok {
		r.mu.RUnlock()
		return pool, nil
	}
	r.mu.RUnlock()

	// Open outside the lock: NewPool blocks on a network/file ping and
	// must not stall other callers.
	pool, err := NewPool(dsn)
	if err != nil {
		return nil, fmt.Errorf("open database %q: %w", label, err)
	}

	r.mu.Lock()
	if existing, ok := r.dbs[label]; ok {
		// Lost the race: another goroutine opened the same label first.
		r.mu.Unlock()
		pool.Close()
		return existing, nil
	}
	r.dbs[label] = pool
	r.names = append(r.names, label)
	r.mu.Unlock()

	return pool, nil
}

// Get returns the pool for an already-connected label, without opening one.
func (r *Registry) Get(name string) (*sql.DB, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	db, ok := r.dbs[name]
	return db, ok
}

// Names returns the labels of databases connected so far in this session.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, len(r.names))
	copy(out, r.names)
	return out
}

// Close closes all pools opened during this session.
func (r *Registry) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, db := range r.dbs {
		db.Close()
	}
}
