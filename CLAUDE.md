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
./fbmcp "user:password@servername/path/to/database.fdb"
./fbmcp --port 9090 "host/path/to/database.fdb"  # default creds: sysdba:masterkey
```

## Architecture

This is a Firebird database **MCP (Model Context Protocol) server** over HTTP (not stdio). MCP clients POST JSON-RPC 2.0 requests to `POST /mcp`. A `GET /health` endpoint pings the DB.

```
cmd/fbmcp/main.go          entry point: parses flags, creates pool, starts HTTP server
internal/mcp/
  server.go                HTTP handler → dispatch() switch on method
  protocol.go              JSON-RPC types, tool input schemas (executeSQLSchema, searchObjectsSchema)
  handlers.go              MCP methods: initialize, tools/list, tools/call
internal/tools/
  execute_sql.go           splits on ";", detects SELECT/WITH vs DML, returns StatementResult[]
  search_objects.go        routes by object_type → firebird query builders, assembles nested results
internal/db/
  pool.go                  sql.DB with conn limits; applyDSNDefaults injects sysdba:masterkey
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
