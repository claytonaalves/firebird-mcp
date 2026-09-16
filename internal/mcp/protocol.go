package mcp

import "encoding/json"

type JSONRPCRequest struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type ToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

const (
	ErrParse          = -32700
	ErrInvalidRequest = -32600
	ErrMethodNotFound = -32601
	ErrInvalidParams  = -32602
	ErrInternal       = -32603
)

var listDatabasesSchema = json.RawMessage(`{
  "type": "object",
  "properties": {}
}`)

var executeSQLSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "database": {
      "type": "string",
      "description": "Database path or DSN to query, in the form host/path/to/db.fdb or user:password@host/path/to/db.fdb (defaults to sysdba:masterkey). On Windows, the path must use the format \"localhost/c:\\\\ello\\\\dados\\\\TESTE.ELLO\" (host/database_path, with backslashes) — not a bare Windows path like \"c:\\\\ello\\\\dados\\\\TESTE.ELLO\". Required — the server has no preconfigured database; each call must specify one. Use list_databases to see databases already connected in this session."
    },
    "sql": {
      "type": "string",
      "description": "SQL statement(s) separated by semicolons"
    }
  },
  "required": ["sql", "database"]
}`)

var searchObjectsSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "database": {
      "type": "string",
      "description": "Database path or DSN to query, in the form host/path/to/db.fdb or user:password@host/path/to/db.fdb (defaults to sysdba:masterkey). On Windows, the path must use the format \"localhost/c:\\\\ello\\\\dados\\\\TESTE.ELLO\" (host/database_path, with backslashes) — not a bare Windows path like \"c:\\\\ello\\\\dados\\\\TESTE.ELLO\". Required — the server has no preconfigured database; each call must specify one. Use list_databases to see databases already connected in this session."
    },
    "object_type": {
      "type": "string",
      "enum": ["table", "column", "procedure", "function", "view"],
      "description": "Type of database object to search"
    },
    "pattern": {
      "type": "string",
      "description": "SQL LIKE pattern (% for wildcard, _ for single char). Defaults to %",
      "default": "%"
    },
    "detail_level": {
      "type": "string",
      "enum": ["names", "summary", "full"],
      "description": "Level of detail to return. Defaults to names",
      "default": "names"
    },
    "limit": {
      "type": "integer",
      "minimum": 1,
      "maximum": 1000,
      "description": "Maximum results to return. Defaults to 100",
      "default": 100
    }
  },
  "required": ["object_type", "database"]
}`)
