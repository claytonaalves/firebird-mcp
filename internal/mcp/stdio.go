package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"sync"
)

// ServeStdio reads newline-delimited JSON-RPC requests from stdin and writes
// responses to stdout, per the MCP stdio transport framing.
//
// Each request is handled in its own goroutine so a slow query never blocks
// later requests (or the notifications/cancelled that would abort it).
// Responses are serialized through a mutex.
func (s *Server) ServeStdio(ctx context.Context) error {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	enc := json.NewEncoder(os.Stdout)
	var writeMu sync.Mutex
	write := func(resp JSONRPCResponse) {
		writeMu.Lock()
		defer writeMu.Unlock()
		if err := enc.Encode(resp); err != nil {
			slog.Error("encode response", "err", err)
		}
	}

	var (
		inflightMu sync.Mutex
		inflight   = map[string]context.CancelFunc{}
		wg         sync.WaitGroup
	)

	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		if len(line) == 0 {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			write(JSONRPCResponse{Jsonrpc: "2.0", Error: &RPCError{Code: ErrParse, Message: "parse error"}})
			continue
		}

		if req.Jsonrpc != "2.0" {
			write(JSONRPCResponse{Jsonrpc: "2.0", ID: req.ID, Error: &RPCError{Code: ErrInvalidRequest, Message: "invalid jsonrpc version"}})
			continue
		}

		if req.Method == "notifications/cancelled" {
			var p struct {
				RequestID json.RawMessage `json:"requestId"`
			}
			if json.Unmarshal(req.Params, &p) == nil {
				inflightMu.Lock()
				cancel := inflight[string(p.RequestID)]
				inflightMu.Unlock()
				if cancel != nil {
					slog.Info("request cancelled by client", "id", string(p.RequestID))
					cancel()
				}
			}
			continue
		}

		reqCtx, cancel := context.WithCancel(ctx)
		key := string(req.ID)
		if req.ID != nil {
			inflightMu.Lock()
			inflight[key] = cancel
			inflightMu.Unlock()
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer cancel()
			result, rpcErr := s.dispatch(reqCtx, req.Method, req.Params)
			if req.ID != nil {
				inflightMu.Lock()
				delete(inflight, key)
				inflightMu.Unlock()
			}

			// notifications (no id) expect no response
			if req.ID == nil && rpcErr == nil {
				return
			}
			// a cancelled request gets no response, per the MCP spec
			if reqCtx.Err() == context.Canceled && ctx.Err() == nil {
				return
			}
			write(JSONRPCResponse{Jsonrpc: "2.0", ID: req.ID, Result: result, Error: rpcErr})
		}()
	}

	// stdin closed: let in-flight requests finish (or be cancelled by ctx)
	wg.Wait()

	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}
