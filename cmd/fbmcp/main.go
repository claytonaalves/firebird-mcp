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

	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "fbmcp no longer takes a database path as a startup argument.\n")
		fmt.Fprintf(os.Stderr, "Pass the database path/DSN per call via the \"database\" argument of execute_sql/search_objects instead.\n")
		os.Exit(1)
	}

	registry := db.NewRegistry()
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
