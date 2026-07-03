package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
)

// ServeStdio reads newline-delimited JSON-RPC requests from stdin and writes
// responses to stdout, per the MCP stdio transport framing.
func (s *Server) ServeStdio(ctx context.Context) error {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	enc := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			if encErr := enc.Encode(JSONRPCResponse{Jsonrpc: "2.0", Error: &RPCError{Code: ErrParse, Message: "parse error"}}); encErr != nil {
				slog.Error("encode response", "err", encErr)
			}
			continue
		}

		if req.Jsonrpc != "2.0" {
			if encErr := enc.Encode(JSONRPCResponse{Jsonrpc: "2.0", ID: req.ID, Error: &RPCError{Code: ErrInvalidRequest, Message: "invalid jsonrpc version"}}); encErr != nil {
				slog.Error("encode response", "err", encErr)
			}
			continue
		}

		result, rpcErr := s.dispatch(ctx, req.Method, req.Params)

		// notifications (no id) expect no response
		if req.ID == nil && rpcErr == nil {
			continue
		}

		resp := JSONRPCResponse{Jsonrpc: "2.0", ID: req.ID, Result: result, Error: rpcErr}
		if err := enc.Encode(resp); err != nil {
			slog.Error("encode response", "err", err)
		}
	}

	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}
