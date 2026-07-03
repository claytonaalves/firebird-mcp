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
	server := flag.Bool("server", false, "serve over HTTP instead of stdio")
	port := flag.String("port", "8080", "port to listen on (with --server)")
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "Usage: fbmcp [--server] [--port PORT] [name=]DSN [[name=]DSN ...]\n")
		fmt.Fprintf(os.Stderr, "  DSN: [user[:password]@]host/path/to/database.fdb\n")
		fmt.Fprintf(os.Stderr, "  name: optional label (default: \"default\")\n")
		fmt.Fprintf(os.Stderr, "  --server: serve over HTTP (default: stdio)\n")
		os.Exit(1)
	}

	registry, err := db.NewRegistry(flag.Args())
	if err != nil {
		slog.Error("connect to database", "err", err)
		os.Exit(1)
	}
	defer registry.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	mcpServer := mcp.NewServer(registry)

	if !*server {
		if err := mcpServer.ServeStdio(ctx); err != nil {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
		return
	}

	srv := &http.Server{
		Addr:    ":" + *port,
		Handler: mcpServer,
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
