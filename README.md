<div align="center">

# <img src="./docs/logo/logo.svg" width="30px" height="auto" alt="" /> WhoDB

### A lightweight, self-hosted workspace for your databases

[![Release workflow](https://img.shields.io/github/actions/workflow/status/clidey/whodb/release-ce.yml?branch=main)](https://github.com/clidey/whodb/actions/workflows/release-ce.yml)
[![Release version](https://img.shields.io/github/v/release/clidey/whodb)](https://github.com/clidey/whodb/releases)
![Release date](https://img.shields.io/github/release-date/clidey/whodb)
![Docker Pulls](https://img.shields.io/docker/pulls/clidey/whodb?label=downloads)
![Docker size](https://img.shields.io/docker/image-size/clidey/whodb/latest)

![Commits per month](https://img.shields.io/github/commit-activity/m/clidey/whodb)
![Last commit](https://img.shields.io/github/last-commit/clidey/whodb)
![Contributors](https://img.shields.io/github/contributors/clidey/whodb)
![Closed issues](https://img.shields.io/github/issues-closed/clidey/whodb)
![Closed PRs](https://img.shields.io/github/issues-pr-closed/clidey/whodb)

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![GitHub stars](https://img.shields.io/github/stars/clidey/whodb?style=social)](https://github.com/clidey/whodb/stargazers)
![Go](https://img.shields.io/badge/language-Go-00ADD8?logo=go&logoColor=white)
![TypeScript](https://img.shields.io/badge/language-TypeScript-3178c6?logo=typescript&logoColor=white)

## Available on

[![Docker](https://img.shields.io/badge/Docker-available-brightgreen)](https://hub.docker.com/r/clidey/whodb)
[![Windows](https://img.shields.io/badge/Windows-available-brightgreen)](https://apps.microsoft.com/detail/9pftx5bv4ds6)
[![macOS](https://img.shields.io/badge/macOS-available-brightgreen)](https://apps.apple.com/app/whodb/id6754566536)
[![Snap](https://img.shields.io/badge/Snap-available-brightgreen)](https://snapcraft.io/whodb)
[![CLI](https://img.shields.io/badge/CLI-available-brightgreen)](./cli/README.md)

[Quick start](#quick-start) · [Documentation](https://docs.whodb.com/) · [Live demo](https://demo.whodb.com/) · [Community](https://github.com/clidey/whodb/discussions)

</div>

<p align="center">
  <img src="./docs/images/06-storage-unit-list-with-sidebar.png" alt="WhoDB showing a database table" width="100%" />
</p>

WhoDB gives you one place to explore your databases, edit data, run queries, and understand how a schema fits together. It runs in the browser, is easy to self-host, and is available as a desktop app or terminal CLI too.

Use it when you want to inspect a local database, help a teammate explore unfamiliar data, or work without installing a heavyweight database client. AI features are optional: connect Ollama, OpenAI, Anthropic, LM Studio, or another OpenAI-compatible provider if you want to ask questions in plain English.

## Quick start

Run WhoDB with Docker:

```bash
docker run --rm -it -p 8080:8080 clidey/whodb
```

Then open [http://localhost:8080](http://localhost:8080) and enter your database connection details.

Want to look around first? Try the [live demo](https://demo.whodb.com/) or watch the [video walkthrough](https://youtu.be/hnAQcYYzcLo).

## What you can do

- **Browse and edit data** in a spreadsheet-style grid with sorting, filtering, pagination, and inline editing.
- **Understand a schema visually** with an interactive graph of tables and relationships.
- **Work through queries in a scratchpad** with multiple cells, autocomplete, history, and results kept alongside each query.
- **Move data in and out** with imports, exports, and mock-data generation for development and testing.
- **Ask questions in plain English** using a local or hosted AI provider that you choose.
- **Work from your terminal** through the WhoDB CLI and its MCP server.

<table>
<tr>
<td width="50%">
  <img src="./docs/images/09-data-view-users-table.png" alt="Browsing rows in the WhoDB data grid" width="100%" />
</td>
<td width="50%">
  <img src="./docs/images/24-graph-view-schema-topology.png" alt="Exploring database relationships in the WhoDB schema graph" width="100%" />
</td>
</tr>
<tr>
<td align="center"><sub>Browse, filter, and edit data</sub></td>
<td align="center"><sub>Follow relationships through the schema graph</sub></td>
</tr>
</table>

### Supported databases

WhoDB Community supports:

- PostgreSQL, CockroachDB, YugabyteDB, and QuestDB
- MySQL, MariaDB, and TiDB
- SQLite and DuckDB
- MongoDB and FerretDB
- Redis, Valkey, and Dragonfly
- Elasticsearch and OpenSearch
- ClickHouse and Memcached

Support varies by database because not every system has the same concepts or capabilities. The connection screen shows the options available for each source.

## Installation options

### Docker with persistent sessions

The one-line Docker command is ideal for trying WhoDB. To keep encrypted login sessions when the container is replaced, first generate a key and save it somewhere secure:

```bash
openssl rand -hex 32
```

Then mount `/data` and reuse that key whenever you start the container:

```bash
docker run -it -p 8080:8080 \
  -v whodb-data:/data \
  -e WHODB_ENCRYPTION_KEY=your_saved_64_character_hex_key \
  clidey/whodb
```

Keep that key somewhere safe. Changing it invalidates existing sessions. If WhoDB is served through an HTTPS reverse proxy, also set `WHODB_SECURE=true` so browsers only send the session cookie over HTTPS.

See the [documentation](https://docs.whodb.com/) for Docker Compose, connection profiles, SSL, AI providers, and other configuration options.

### Desktop

- [macOS](https://apps.apple.com/app/whodb/id6754566536)
- [Windows](https://apps.microsoft.com/detail/9pftx5bv4ds6)
- [Snap](https://snapcraft.io/whodb)

### CLI and MCP server

The CLI includes an interactive terminal UI and an MCP server for AI tools:

```bash
# macOS and Linux
curl -fsSL https://raw.githubusercontent.com/clidey/whodb/main/cli/install/install.sh | bash

# or install with npm
npm install -g @clidey/whodb-cli
```

```bash
whodb-cli             # open the terminal UI
whodb-cli mcp serve   # start the MCP server
```

See the [CLI guide](./cli/README.md) for connection examples and the full command reference.

## AI providers

You can add a hosted AI provider directly from WhoDB—no backend configuration or restart is required. Open Chat, choose **Add Provider** from the provider menu, then select OpenAI, Anthropic, or Gemini and enter your API key. WhoDB will fetch the available models from that provider.

Ollama and LM Studio are available as local options in WhoDB Community. By default, the backend looks for Ollama at `localhost:11434` and LM Studio at `localhost:1234/v1`, with local addresses adjusted automatically for Docker and WSL. Use `WHODB_OLLAMA_HOST`, `WHODB_OLLAMA_PORT`, or `WHODB_LMSTUDIO_BASE_URL` only when those defaults are not suitable for your setup.

### Optional backend provider configuration

Environment variables let deployment administrators declare providers when the server starts. This is useful for preconfiguring OpenAI or Anthropic, changing provider endpoints, or adding an OpenAI-compatible service for everyone using that deployment:

- `WHODB_OPENAI_*` for OpenAI
- `WHODB_ANTHROPIC_*` for Anthropic
- `WHODB_OLLAMA_*` for Ollama connection settings
- `WHODB_LMSTUDIO_*` for LM Studio connection settings
- `WHODB_AI_GENERIC_<ID>_*` for OpenAI-compatible providers

See the [installation guide](./docs/installation.mdx#ai-providers) for the complete environment variable list and the [AI provider guide](./docs/ai/setup-providers.mdx) for setup examples.

## WhoDB Community and WhoDB Platform

This repository contains **WhoDB Community**, the Apache-2.0-licensed database workspace described above. It is free to self-host and is the best place to start if you want to explore and work with databases.

**WhoDB Platform** is the commercial, self-hosted edition for organizations that need shared projects, more data sources, SSO, fine-grained access controls, audit logs, pipelines, reporting, and internal apps. You can read the [WhoDB overview](https://whodb.com/) or compare plans on the [pricing page](https://whodb.com/pricing/).

## Development

WhoDB has a Go backend and a React/TypeScript frontend. For local development, run them in separate terminals.

Requirements:

- Go
- Node.js and pnpm

Start the backend:

```bash
cd core
go run ./cmd/whodb
```

Start the frontend:

```bash
cd frontend
pnpm install
pnpm start
```

The frontend opens at [http://localhost:3000](http://localhost:3000) and talks to the backend on port `8080`. See [BUILD_AND_RUN.md](./BUILD_AND_RUN.md) for generation and build commands.

## Contributing

Bug reports, feature ideas, documentation improvements, and code contributions are all welcome.

- [Open an issue](https://github.com/clidey/whodb/issues) for a bug or concrete request.
- [Start a discussion](https://github.com/clidey/whodb/discussions) for questions and ideas.
- Read [CONTRIBUTING.md](./CONTRIBUTING.md) before sending a pull request.

## More screenshots

<details>
<summary>See WhoDB in action</summary>

### Query scratchpad

<img src="./docs/images/27-scratchpad-main-view.png" alt="Writing queries in the WhoDB scratchpad" width="100%" />

### Add and edit records

<img src="./docs/images/11-data-view-add-row-dialog.png" alt="Adding a row in WhoDB" width="100%" />

### Build filters visually

<img src="./docs/images/16-data-view-where-conditions-popover.png" alt="Building WHERE conditions in WhoDB" width="100%" />

### Export data

<img src="./docs/images/20-data-view-export-dialog.png" alt="Exporting data from WhoDB" width="100%" />

</details>

## Support

- [Documentation](https://docs.whodb.com/)
- [GitHub Issues](https://github.com/clidey/whodb/issues)
- [support@clidey.com](mailto:support@clidey.com)

WhoDB is licensed under the [Apache License 2.0](./LICENSE).

<div align="center">

If WhoDB saves you time, consider [giving the project a star](https://github.com/clidey/whodb/stargazers).

</div>
