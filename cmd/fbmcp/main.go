package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"

	"github.com/claytonaalves/firebird-mcp/internal/db"
	"github.com/claytonaalves/firebird-mcp/internal/mcp"
)

func main() {
	port := flag.String("port", "8080", "port to listen on")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Usage: fbmcp [--port PORT] DSN\n")
		fmt.Fprintf(os.Stderr, "  DSN: [user[:password]@]host/path/to/database.fdb\n")
		os.Exit(1)
	}

	pool, err := db.NewPool(flag.Arg(0))
	if err != nil {
		slog.Error("connect to database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	srv := &http.Server{
		Addr:    ":" + *port,
		Handler: mcp.NewServer(pool),
	}

	go func() {
		<-ctx.Done()
		slog.Info("shutting down")
		srv.Shutdown(context.Background())
	}()

	slog.Info("firebird-mcp listening", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
