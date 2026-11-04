package mcp

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/claytonaalves/firebird-mcp/internal/db"
)

// DefaultQueryTimeout bounds each tools/call. When it fires the context is
// cancelled and the firebirdsql driver sends op_cancel, so the statement
// stops running on the Firebird server too. Zero disables the timeout.
const DefaultQueryTimeout = 60 * time.Second

type Server struct {
	registry     *db.Registry
	mux          *http.ServeMux
	queryTimeout time.Duration
}

func NewServer(registry *db.Registry, queryTimeout time.Duration) *Server {
	s := &Server{registry: registry, mux: http.NewServeMux(), queryTimeout: queryTimeout}
	s.mux.HandleFunc("POST /mcp", s.handleMCP)
	s.mux.HandleFunc("GET /health", s.handleHealth)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, nil, ErrParse, "parse error")
		return
	}

	if req.Jsonrpc != "2.0" {
		writeError(w, req.ID, ErrInvalidRequest, "invalid jsonrpc version")
		return
	}

	result, rpcErr := s.dispatch(r.Context(), req.Method, req.Params)
	if rpcErr != nil {
		resp := JSONRPCResponse{Jsonrpc: "2.0", ID: req.ID, Error: rpcErr}
		json.NewEncoder(w).Encode(resp)
		return
	}

	// notifications (no id) expect no response
	if req.ID == nil && result == nil {
		return
	}

	resp := JSONRPCResponse{Jsonrpc: "2.0", ID: req.ID, Result: result}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("encode response", "err", err)
	}
}

func (s *Server) dispatch(ctx context.Context, method string, params json.RawMessage) (any, *RPCError) {
	switch method {
	case "initialize":
		return handleInitialize(ctx, s.registry, params)
	case "notifications/initialized":
		return nil, nil
	case "tools/list":
		return handleToolsList(ctx, s.registry, params)
	case "tools/call":
		if s.queryTimeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, s.queryTimeout)
			defer cancel()
		}
		return handleToolsCall(ctx, s.registry, params)
	default:
		return nil, &RPCError{Code: ErrMethodNotFound, Message: "method not found: " + method}
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	type dbStatus struct {
		Status string `json:"status"`
		Error  string `json:"error,omitempty"`
	}
	statuses := map[string]dbStatus{}
	allOK := true
	for _, name := range s.registry.Names() {
		pool, _ := s.registry.Get(name)
		if err := pool.PingContext(r.Context()); err != nil {
			statuses[name] = dbStatus{Status: "error", Error: err.Error()}
			allOK = false
		} else {
			statuses[name] = dbStatus{Status: "ok"}
		}
	}
	if !allOK {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	json.NewEncoder(w).Encode(map[string]any{"databases": statuses})
}

func writeError(w http.ResponseWriter, id json.RawMessage, code int, msg string) {
	resp := JSONRPCResponse{
		Jsonrpc: "2.0",
		ID:      id,
		Error:   &RPCError{Code: code, Message: msg},
	}
	json.NewEncoder(w).Encode(resp)
}
