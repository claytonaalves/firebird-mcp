# Firebird MCP

## Commands

```bash
make build    # builds ./fbmcp binary
make install  # go install
make tidy     # go mod tidy
go test ./... # run tests (none yet)
```

Run the server:
```bash
./fbmcp                              # stdio transport (default); no database configured at startup
./fbmcp --server --port 9090         # HTTP transport
```

The server has no preconfigured database — every `execute_sql`/`search_objects` call must pass a `database` argument with a full DSN/path, e.g. `user:password@servername/path/to/database.fdb` or just `host/path/to/database.fdb` (defaults to `sysdba:masterkey`). The connection is opened on first use and cached for the rest of the session (see `internal/db/pool.go`'s `Registry.Resolve`).

## Architecture

This is a Firebird database **MCP (Model Context Protocol) server**. It defaults to the **stdio** transport (newline-delimited JSON-RPC 2.0 over stdin/stdout); pass `--server` to serve over HTTP instead, where clients POST JSON-RPC 2.0 requests to `POST /mcp` and a `GET /health` endpoint pings whatever databases have been connected so far.

```
cmd/fbmcp/main.go          entry point: parses flags, runs stdio loop or starts HTTP server
internal/mcp/
  server.go                HTTP handler → dispatch() switch on method
  stdio.go                 stdio transport loop → same dispatch()
  protocol.go              JSON-RPC types, tool input schemas (executeSQLSchema, searchObjectsSchema)
  handlers.go              MCP methods: initialize, tools/list, tools/call
internal/tools/
  execute_sql.go           splits on ";", detects SELECT/WITH vs DML, returns StatementResult[]
  search_objects.go        routes by object_type → firebird query builders, assembles nested results
internal/db/
  pool.go                  Registry: lazy connection cache keyed by sanitized DSN (Resolve); applyDSNDefaults injects sysdba:masterkey
  query.go                 RunQuery (→ cols+rows) and RunExec (→ rows affected)
internal/firebird/
  schema.go                all Firebird RDB$ system-table SQL; returns QuerySpec{SQL, Args}
```

## Key design points

**Two MCP tools:**
- `execute_sql` — runs SQL statements (semicolon-separated); SELECT/WITH goes through `RunQuery`, everything else through `RunExec`
- `search_objects` — progressive disclosure over `object_type` × `detail_level` (names/summary/full); schema queries live in `internal/firebird/schema.go`

**Adding a new MCP tool** requires changes in three places:
1. `internal/mcp/protocol.go` — add the input schema (`json.RawMessage`)
2. `internal/mcp/handlers.go` — add the tool to `handleToolsList` and a case in `handleToolsCall`
3. `internal/tools/` — implement the tool function

**Firebird query layer:** `internal/firebird/schema.go` builds all queries against Firebird's `RDB$` system catalog. `limit` is interpolated directly into SQL (not a bind param) because Firebird's `ROWS 1 TO N` clause doesn't accept parameters. Table names in `fetchRowCounts` are also interpolated — they come from `RDB$RELATIONS`, not user input.

**Result assembly:** `search_objects` `"full"` detail level fetches flat join rows and assembles nested structures in Go (`groupTableColumns`, `mergeIndexes`). `rowsToMaps` strips nil and blank fields for token efficiency.
