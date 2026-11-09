# Firebird MCP

A Firebird database MCP server. Allows MCP-compatible clients to connect to and explore Firebird databases. Uses the stdio transport by default (HTTP optional with `--server`).

## Installation

Build (produces `fbmcp` for Linux and `fbmcp.exe` for Windows):

    make build

### Linux

Register the server in Claude Code:

    claude mcp add fbmcp -- /home/youruser/path/to/firebird-mcp/fbmcp

Add `--scope user` (before the `--`) to make it available in every project.

### Windows

Copy `fbmcp.exe` to a fixed location, e.g. `C:\bin\fbmcp.exe`, then run:

    claude mcp add fbmcp -- C:\bin\fbmcp.exe

Check that it is registered with `claude mcp list` (or `/mcp` inside Claude Code).

## Usage

The server has no preconfigured database. Every `execute_sql` / `search_objects` call must pass a `database` argument with a DSN:

    user:password@host/path/to/db.fdb

Credentials are optional and default to `sysdba:masterkey`:

    host/path/to/db.fdb

The connection is opened on first use and cached for the rest of the session.

To serve over HTTP instead of stdio (clients POST JSON-RPC to `/mcp`; `GET /health` pings connected databases):

    ./fbmcp --server --port 9090

## MCP Tools

### `execute_sql`

Execute SQL queries and statements on a Firebird database.

- **Single statement** — SELECT, INSERT, UPDATE, DELETE, or DDL
- **Multiple statements** — separate with semicolons (`;`)

Examples:

```sql
SELECT * FROM users WHERE status = 'active';
```

```sql
INSERT INTO users (name, email) VALUES ('Alice', 'alice@example.com');
INSERT INTO users (name, email) VALUES ('Bob', 'bob@example.com');
```

For read-only access, connect with a database user that has read-only permissions.

---

### `search_objects`

Search and list database objects with pattern matching. Implements progressive disclosure to minimize token usage.

**Parameters:**

| Parameter | Required | Description |
|---|---|---|
| `object_type` | yes | `table`, `column`, `procedure`, `function`, or `view` |
| `pattern` | no | SQL LIKE pattern (`%` wildcard, `_` single char). Defaults to `%` |
| `detail_level` | no | `names` (default), `summary`, or `full` |
| `limit` | no | Max results (default: 100, max: 1000) |

**Detail levels:**

`names` — minimal tokens, just names:
```json
{ "count": 3, "results": [{ "name": "users" }, { "name": "user_profiles" }] }
```

`summary` — adds metadata (column count, row count, comment):
```json
{ "count": 1, "results": [{ "name": "users", "column_count": 8, "row_count": 1523, "comment": "Application users" }] }
```

`full` — complete structure with columns, indexes, and comments.

**Pattern syntax** (SQL LIKE):

```
user%    → "users", "user_profiles", "user_sessions"
%_id     → "user_id", "order_id", "product_id"
%user%   → "users", "user_profiles", "app_users"
```

### `list_databases`

Lists all registered databases by name. Used with multi-database configurations.

## Recommended workflow

Start broad, then drill down:

```json
{ "object_type": "table", "detail_level": "names" }
```

```json
{ "object_type": "table", "pattern": "users", "detail_level": "full" }
```

Always start with `detail_level: "names"` and only request `"summary"` or `"full"` when you need more information. This reduces token usage by 90–99% compared to loading all objects upfront.
