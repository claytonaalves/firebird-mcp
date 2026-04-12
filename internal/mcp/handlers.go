package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/claytonaalves/firebird-mcp/internal/tools"
)

type initializeResult struct {
	ProtocolVersion string `json:"protocolVersion"`
	Capabilities    struct {
		Tools struct{} `json:"tools"`
	} `json:"capabilities"`
	ServerInfo struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
}

type toolsListResult struct {
	Tools []ToolDef `json:"tools"`
}

type toolsCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type toolsCallResult struct {
	Content []TextContent `json:"content"`
}

func handleInitialize(_ context.Context, _ *sql.DB, _ json.RawMessage) (any, *RPCError) {
	var r initializeResult
	r.ProtocolVersion = "2024-11-05"
	r.ServerInfo.Name = "firebird-mcp"
	r.ServerInfo.Version = "1.0.0"
	return r, nil
}

func handleToolsList(_ context.Context, _ *sql.DB, _ json.RawMessage) (any, *RPCError) {
	return toolsListResult{
		Tools: []ToolDef{
			{
				Name:        "execute_sql",
				Description: "Execute SQL queries and statements on the Firebird database. Supports single and multiple statements separated by semicolons.",
				InputSchema: executeSQLSchema,
			},
			{
				Name:        "search_objects",
				Description: "Search and list database objects (tables, columns, procedures, functions, views) with pattern matching and progressive disclosure.",
				InputSchema: searchObjectsSchema,
			},
		},
	}, nil
}

func handleToolsCall(ctx context.Context, db *sql.DB, params json.RawMessage) (any, *RPCError) {
	var p toolsCallParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &RPCError{Code: ErrInvalidParams, Message: "invalid tools/call params"}
	}

	var (
		result any
		err    error
	)
	switch p.Name {
	case "execute_sql":
		result, err = tools.ExecuteSQL(ctx, db, p.Arguments)
	case "search_objects":
		result, err = tools.SearchObjects(ctx, db, p.Arguments)
	default:
		return nil, &RPCError{Code: ErrMethodNotFound, Message: fmt.Sprintf("unknown tool: %s", p.Name)}
	}

	if err != nil {
		return nil, &RPCError{Code: ErrInternal, Message: err.Error()}
	}

	text, err := json.Marshal(result)
	if err != nil {
		return nil, &RPCError{Code: ErrInternal, Message: "marshal result: " + err.Error()}
	}

	return toolsCallResult{
		Content: []TextContent{{Type: "text", Text: string(text)}},
	}, nil
}
