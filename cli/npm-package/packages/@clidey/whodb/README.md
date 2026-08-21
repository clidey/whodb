# @clidey/whodb

The [WhoDB](https://whodb.com) CLI — an interactive, production-ready command-line
interface for navigating SQL and NoSQL databases, with a TUI, programmatic
commands, and an MCP server for AI assistants.

This is the primary namespaced package for installing WhoDB's CLI:

```bash
npm install -g @clidey/whodb
whodb --help
```

## What this package is

This is a thin wrapper: all the real work — the launcher, the platform-specific
binaries, and the BAML runtime — lives in
[`@clidey/whodb-cli`](https://www.npmjs.com/package/@clidey/whodb-cli), which
this package depends on and forwards to. Installing this package gives you
the `whodb` command.

> Renamed from `whodb-cli`: the command used to be `whodb-cli`. If you're
> upgrading, `@clidey/whodb-cli` still ships a deprecated `whodb-cli` binary
> that prints a warning and forwards to `whodb` — update scripts and configs
> to use `whodb` going forward.

## Features

- **Interactive TUI** — terminal UI with split-pane layouts, themes, and an
  SQL editor with schema-aware autocomplete
- **Multi-database support** — PostgreSQL, MySQL/MariaDB, SQLite, MongoDB,
  Redis, ClickHouse, Elasticsearch
- **Programmatic mode** — JSON/NDJSON/CSV/plain output for scripting and CI
- **Import/export** — CSV and Excel, plus FK-aware mock data generation
- **Schema tools** — ERD graph output, EXPLAIN plans, schema diff between
  connections
- **MCP server** — expose database access to AI assistants (Claude, Cursor,
  and others) with read-only and confirm-writes safety modes
- **Assistant integration installer** — bundled skills, agents, and MCP
  configs for popular coding assistants

## Usage

```bash
# Interactive TUI
whodb

# Run a query
whodb query "SELECT * FROM users LIMIT 10" --connection mydb

# Start as an MCP server
whodb mcp serve
```

Or without installing:

```bash
npx @clidey/whodb
```

## Documentation

Full CLI documentation, command reference, and MCP setup:
https://github.com/clidey/whodb/tree/main/cli#readme
