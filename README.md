# MySQL MCP Server (Go)

A Model Context Protocol (MCP) server that provides MySQL database access to AI assistants like Claude.

## Features

- **Multi-connection support**: Configure multiple database connections (production, staging, local, etc.)
- **Read-only mode**: Configurable per connection to prevent accidental writes
- **Row limits**: Configurable max rows per connection to prevent large result sets
- **Safety features**: Blocks dangerous operations (DROP, ALTER, TRUNCATE, etc.)
- **Query timeout**: 30-second timeout prevents long-running queries

## Installation

### Build from source

```bash
cd mysql-golang-mcp
go build -o mysql-mcp
```

## Configuration

Create a `config.json` file with your database connections:

```json
{
  "connections": {
    "production": {
      "host": "prod-db.example.com",
      "port": 3306,
      "user": "reader",
      "password": "secret",
      "database": "app_db",
      "read_only": true,
      "max_rows": 1000
    },
    "staging": {
      "host": "staging-db.example.com",
      "port": 3306,
      "user": "admin",
      "password": "secret",
      "database": "app_db",
      "read_only": false,
      "max_rows": 5000
    }
  }
}
```

### Configuration Options

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `host` | Yes | - | MySQL server hostname |
| `port` | No | 3306 | MySQL server port |
| `user` | Yes | - | Database username |
| `password` | No | "" | Database password |
| `database` | Yes | - | Default database name |
| `read_only` | No | false | Only allow SELECT/SHOW/DESCRIBE/EXPLAIN |
| `max_rows` | No | 1000 | Maximum rows to return per query |

### Config File Location

The config file path is determined in this order:

1. `--config` command line flag
2. `MYSQL_MCP_CONFIG` environment variable
3. `./config.json` (default)

## Claude Code Integration

Add to your Claude Code MCP configuration (`~/.claude/claude_desktop_config.json`):

### Using command line flag

```json
{
  "mcpServers": {
    "mysql": {
      "command": "/path/to/mysql-mcp",
      "args": ["--config", "/path/to/config.json"]
    }
  }
}
```

### Using environment variable

```json
{
  "mcpServers": {
    "mysql": {
      "command": "/path/to/mysql-mcp",
      "env": {
        "MYSQL_MCP_CONFIG": "/path/to/config.json"
      }
    }
  }
}
```

## Available Tools

### Query Tools (Segregated by Type)

These tools allow MCP clients to selectively auto-accept based on risk level:

| Tool | SQL Types | Risk | Auto-Accept Safe? |
|------|-----------|------|-------------------|
| `mysql_select` | SELECT | Low | Yes |
| `mysql_insert` | INSERT | Medium | Maybe |
| `mysql_update` | UPDATE | High | No |
| `mysql_delete` | DELETE | High | No |
| `mysql_alter` | ALTER TABLE | High | No |
| `mysql_execute` | INSERT/UPDATE/DELETE | High | No |
| `mysql_execute_unsafe` | ANY | CRITICAL | Never |
| `mysql_query` | Any (deprecated) | High | No |

### `mysql_select`

Execute a SELECT query. **Safe for auto-accept.**

**Parameters**:
- `connection` (required): Named connection to use
- `sql` (required): The SELECT query to execute

**Example**:
```json
{
  "connection": "production",
  "sql": "SELECT id, name FROM users LIMIT 10"
}
```

### `mysql_insert`

Execute an INSERT query. **Medium risk.**

**Parameters**:
- `connection` (required): Named connection to use
- `sql` (required): The INSERT query to execute

**Example**:
```json
{
  "connection": "staging",
  "sql": "INSERT INTO logs (message) VALUES ('test')"
}
```

### `mysql_update`

Execute an UPDATE query. **High risk - do not auto-accept.**

**Parameters**:
- `connection` (required): Named connection to use
- `sql` (required): The UPDATE query to execute

### `mysql_delete`

Execute a DELETE query. **High risk - do not auto-accept.**

**Parameters**:
- `connection` (required): Named connection to use
- `sql` (required): The DELETE query to execute

### `mysql_alter`

Execute an ALTER TABLE query. **High risk - do not auto-accept.**

Allows schema modifications like adding columns, indexes, and constraints. Still blocks DROP DATABASE, CREATE DATABASE, GRANT, REVOKE.

**Parameters**:
- `connection` (required): Named connection to use
- `sql` (required): The ALTER query to execute

**Example**:
```json
{
  "connection": "staging",
  "sql": "ALTER TABLE users ADD COLUMN last_login DATETIME"
}
```

### `mysql_execute`

Execute INSERT, UPDATE, or DELETE queries. **High risk - do not auto-accept.**

Combined tool for write operations when you don't want separate tools.

**Parameters**:
- `connection` (required): Named connection to use
- `sql` (required): The INSERT, UPDATE, or DELETE query to execute

### `mysql_execute_unsafe`

⚠️ **CRITICAL RISK - NEVER auto-accept.**

Execute ANY SQL query, bypassing all safety checks.

**Bypasses**:
- Dangerous query blocking (DROP, TRUNCATE, CREATE, GRANT, REVOKE)
- Sensitive query blocking (SHOW GRANTS, mysql.user access)

**Does NOT bypass**:
- Read-only connection restrictions (configuration-based)

**Use cases**:
- Emergency fixes requiring schema changes
- Legitimate operations blocked by safety checks
- Administrative tasks

**Parameters**:
- `connection` (required): Named connection to use
- `sql` (required): The SQL query to execute (any type)

**Response includes**:
- `warning`: Reminder that safety checks were bypassed
- `skipped_check`: What specific checks were skipped

### `mysql_query` (Deprecated)

**Deprecated**: Use the specific tools above instead.

Execute a SQL query against the database.

**Parameters**:
- `connection` (required): Named connection to use
- `sql` (required): SQL query to execute

**Example**:
```json
{
  "connection": "production",
  "sql": "SELECT id, name FROM users LIMIT 10"
}
```

### `list_connections`

List all configured database connections.

**Parameters**: None

**Example response**:
```json
[
  {
    "name": "production",
    "read_only": true
  },
  {
    "name": "staging",
    "read_only": false
  }
]
```

### `list_databases`

List all accessible databases.

**Parameters**:
- `connection` (required): Named connection to use

### `list_tables`

List tables in a database.

**Parameters**:
- `connection` (required): Named connection to use
- `database` (optional): Database name (uses connection default if not provided)

### `describe_table`

Get table schema/structure.

**Parameters**:
- `connection` (required): Named connection to use
- `table` (required): Table name
- `database` (optional): Database name

### `get_indexes`

Get indexes for a table.

**Parameters**:
- `connection` (required): Named connection to use
- `table` (required): Table name
- `database` (optional): Database name

## Safety Features

### Read-Only Mode

When `read_only: true` is set for a connection, only these query types are allowed:
- SELECT
- SHOW
- DESCRIBE / DESC
- EXPLAIN

### Blocked Operations

Even when `read_only: false`, these dangerous operations are blocked:
- DROP
- ALTER
- TRUNCATE
- CREATE
- GRANT
- REVOKE

### Row Limits

Each connection has a configurable `max_rows` limit (default: 1000) to prevent accidentally returning massive result sets.

### Query Timeout

All queries have a 30-second timeout to prevent long-running queries from blocking resources.

## Development

```bash
# Run directly
go run . --config ./config.json

# Build
go build -o mysql-mcp

# Test with JSON-RPC
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | ./mysql-mcp --config ./config.json
```

## License

MIT
