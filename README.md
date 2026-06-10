# Firebird MCP

A Firebird database MCP server. Allows MCP-compatible clients to connect to and explore Firebird databases. Runs as a standalone HTTP server.

## Usage

Single database:

    ./fbmcp host/path/to/db.fdb

With credentials:

    ./fbmcp user:password@host/path/to/db.fdb

Custom port (default: 8080):

    ./fbmcp --port 9090 host/path/to/db.fdb

Multiple databases:

    ./fbmcp sales=host/sales.fdb hr=host/hr.fdb reporting=host/reports.fdb

The LLM can then call `list_databases` to enumerate them, and pass `{"database": "hr", "sql": "..."}` to `execute_sql`.

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
