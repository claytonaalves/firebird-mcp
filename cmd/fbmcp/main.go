package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/claytonaalves/firebird-mcp/internal/db"
	"github.com/claytonaalves/firebird-mcp/internal/mcp"
)

func main() {
	port := flag.String("port", "8080", "port to listen on")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Usage: fbmcp [--port PORT] DSN\n")
		fmt.Fprintf(os.Stderr, "  DSN: user:password@host/path/to/database.fdb\n")
		os.Exit(1)
	}

	pool, err := db.NewPool(flag.Arg(0))
	if err != nil {
		slog.Error("connect to database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	server := mcp.NewServer(pool)
	addr := ":" + *port
	slog.Info("firebird-mcp listening", "addr", addr)
	if err := http.ListenAndServe(addr, server); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
