package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
)

type Server struct {
	db  *sql.DB
	mux *http.ServeMux
}

func NewServer(db *sql.DB) *Server {
	s := &Server{db: db, mux: http.NewServeMux()}
	s.mux.HandleFunc("POST /mcp", s.handleMCP)
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
		return handleInitialize(ctx, s.db, params)
	case "notifications/initialized":
		return nil, nil
	case "tools/list":
		return handleToolsList(ctx, s.db, params)
	case "tools/call":
		return handleToolsCall(ctx, s.db, params)
	default:
		return nil, &RPCError{Code: ErrMethodNotFound, Message: "method not found: " + method}
	}
}

func writeError(w http.ResponseWriter, id json.RawMessage, code int, msg string) {
	resp := JSONRPCResponse{
		Jsonrpc: "2.0",
		ID:      id,
		Error:   &RPCError{Code: code, Message: msg},
	}
	json.NewEncoder(w).Encode(resp)
}
