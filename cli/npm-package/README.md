# @clidey/whodb-mcp

WhoDB MCP (Model Context Protocol) server for Claude Code and other AI assistants.

## Installation

```bash
npx @clidey/whodb-mcp
```

Or install globally:

```bash
npm install -g @clidey/whodb-mcp
whodb-mcp
```

## What is this?

This package automatically downloads and runs the WhoDB CLI as an MCP server, enabling AI assistants like Claude to query your databases.

## Configuration

Set up database connections via environment variables:

```bash
export WHODB_PROD_URI="postgres://user:pass@localhost:5432/mydb"
```

## Available Tools

- `whodb_connections` - List available database connections
- `whodb_schemas` - List database schemas
- `whodb_tables` - List tables in a schema
- `whodb_columns` - Describe table columns
- `whodb_query` - Execute SQL queries

## Supported Databases

PostgreSQL, MySQL, MariaDB, SQLite, MongoDB, Redis, ClickHouse, Elasticsearch, and more.

## Links

- [WhoDB Repository](https://github.com/clidey/whodb)
- [Documentation](https://github.com/clidey/whodb/tree/main/cli)

## License

Apache-2.0
