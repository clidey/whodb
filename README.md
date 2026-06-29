<div align="center">

# <img src="./docs/logo/logo.svg" width="30px" height="auto" />  WhoDB

### *The open-source entry point to the WhoDB AI data platform*

<!-- [![Build Status](https://hello.clidey.com/api/flows/status?id=b32257fa-1415-4847-a0f3-e684f5f76608&secret=cd74dbd5-36ec-42f9-b4f0-12ce9fcc762b)](https://clidey.com) -->
![Release workflow](https://img.shields.io/github/actions/workflow/status/clidey/whodb/release-ce.yml?branch=main)
![release version](https://img.shields.io/github/v/release/clidey/whodb)
![release date](https://img.shields.io/github/release-date/clidey/whodb)
![docker pulls](https://img.shields.io/docker/pulls/clidey/whodb)
![release downloads](https://img.shields.io/github/downloads/clidey/whodb/total)
![docker size](https://img.shields.io/docker/image-size/clidey/whodb/latest)

[//]: # ([![E2E Tests]&#40;https://github.com/clidey/whodb/actions/workflows/e2e-ce.yml/badge.svg&#41;]&#40;https://github.com/clidey/whodb/actions/workflows/e2e-ce.yml&#41;)

![Commits per month](https://img.shields.io/github/commit-activity/m/clidey/whodb)
![last commit](https://img.shields.io/github/last-commit/clidey/whodb)
![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)
![contributors](https://img.shields.io/github/contributors/clidey/whodb)
![closed issues](https://img.shields.io/github/issues-closed/clidey/whodb)
![closed PRs](https://img.shields.io/github/issues-pr-closed/clidey/whodb)

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![GitHub Stars](https://img.shields.io/github/stars/clidey/whodb?style=social)](https://github.com/clidey/whodb/stargazers)
![Go](https://img.shields.io/badge/language-Go-00ADD8?logo=go&logoColor=white)
![TypeScript](https://img.shields.io/badge/language-TypeScript-3178c6?logo=typescript&logoColor=white)
[![Go Report Card](https://goreportcard.com/badge/github.com/clidey/whodb/core)](https://goreportcard.com/report/github.com/clidey/whodb/core)

## Available on

[![Docker](https://img.shields.io/badge/Docker-available-brightgreen)](https://hub.docker.com/repository/docker/clidey/whodb)
[![Windows](https://img.shields.io/badge/Windows-available-brightgreen)](https://apps.microsoft.com/detail/9pftx5bv4ds6)
[![macOS](https://img.shields.io/badge/macOS-available-brightgreen)](https://apps.apple.com/app/whodb/id6754566536)
[![Snap](https://img.shields.io/badge/Snap-available-brightgreen)](http://snapcraft.io/whodb)
[![CLI](https://img.shields.io/badge/CLI-available-brightgreen)](./cli/README.md)

[🚀 Quick Start](#quick-start) • [🏢 WhoDB EE Platform](#from-ce-to-the-whodb-ee-platform) • [📖 Documentation](https://docs.whodb.com/) • [🎮 Live Demo](https://demo.whodb.com/login?host=quick-container-491288b0-3138-48fa-93b4-1e730296c0b7.hello.svc.cluster.local&username=user&password=password&database=Adventureworks) • [💬 Community](https://github.com/clidey/whodb/discussions)

</div>

---

<p align="center"><img src="./docs/images/06-storage-unit-list-with-sidebar.png" alt="WhoDB Interface" width="100%" height="auto" /></p>

## 🎯 What is WhoDB?

**WhoDB CE is the open-source data workspace that grows into the WhoDB EE AI platform.**

Built with Go and React, WhoDB CE gives developers a fast, lightweight (<50MB) way to connect to operational data, inspect sources, query with SQL or AI, and understand how data is shaped. It is the first step in the larger WhoDB platform: an AI data layer for sources, ETLs, ontology, governance, and internal apps.

Start with CE when you need a better way to work with databases. Move into WhoDB EE when your team needs governed AI decisions across many systems, not another dashboard.

### Why WhoDB?

<table>
<tr>
<td width="50%">

**🚀 Fast Source Access**
- Instant startup (<1s)
- Real-time query results
- Efficient table virtualization
- 90% less resource usage than traditional tools

**🎨 Clear Operational Context**
- Clean, modern interface
- Spreadsheet-like data grid
- Interactive schema visualization
- No training required

</td>
<td width="50%">

**🤖 AI-Ready Foundation**
- Natural language to SQL
- Talk to your data conversationally
- Supports Ollama, OpenAI, Anthropic, and any OpenAI-compatible provider
- No complex query writing needed

**🔧 Platform Path**
- Multi-database support
- Query history & management
- Mock data generation
- Enterprise platform for more sources, governance, and apps

</td>
</tr>
</table>

## ✨ Start With CE: Source Access, Querying, and Exploration

### 📊 Visual Data Management

<table>
<tr>
<td width="50%">
<img src="./docs/images/09-data-view-users-table.png" alt="Data Grid View" width="100%"/>
</td>
<td width="50%">

**Spreadsheet-Like Data Grid**
- View, edit, and manage data intuitively
- Sort, filter, and search with ease
- Inline editing with real-time updates
- Bulk operations for efficiency

</td>
</tr>
</table>

### 🔍 Interactive Schema Explorer

<table>
<tr>
<td width="50%">

**Visual Schema Topology**
- Interactive graph visualization
- Explore table relationships
- Understand foreign keys instantly
- Pan, zoom, and navigate easily

</td>
<td width="50%">
<img src="./docs/images/24-graph-view-schema-topology.png" alt="Schema Graph" width="100%"/>
</td>
</tr>
</table>

### 💻 Powerful Query Interface

<table>
<tr>
<td width="50%">
<img src="./docs/images/27-scratchpad-main-view.png" alt="Scratchpad" width="100%"/>
</td>
<td width="50%">

**Scratchpad Query Editor**
- Jupyter-like notebook interface
- Syntax highlighting & auto-completion
- Query history & reuse
- Multi-cell organization

</td>
</tr>
</table>

### 🗄️ Multi-Source Foundation

**Community Edition (CE):** PostgreSQL, MySQL, SQLite3, MongoDB, Redis, MariaDB, ElasticSearch

**Enterprise Edition (EE):** All CE databases plus Oracle, SQL Server, DynamoDB, Athena, Snowflake, Cassandra, and more

### 🎯 Advanced Capabilities

- **Mock Data Generation** - Generate realistic test data for development
- **Flexible Export Options** - Export to CSV, Excel, JSON, or SQL
- **Advanced Filtering** - Build complex WHERE conditions visually
- **AI-Powered Queries** - Convert natural language to SQL with Ollama, OpenAI, Anthropic, or any OpenAI-compatible provider

---

## From CE to the WhoDB EE Platform

CE is the sharp wedge: connect to your data quickly, understand it visually, and query it without fighting a heavy desktop client.

WhoDB EE turns that foundation into a platform for faster decisions across the company.

The platform direction is built around:

- **All sources** - Bring operational systems into one governed data layer
- **ETLs** - Move and shape data so teams and AI agents can use it reliably
- **Ontology** - Map raw tables, collections, and events into business objects like customers, accounts, orders, tickets, incidents, invoices, and workflows
- **Governance** - Control who can see, query, change, and act on data
- **Apps** - Build internal operational apps on top of shared source context
- **WhoDB AI Agent** - Ask questions and take action across sources with the right context and permissions

The point is not to make people write SQL faster. The point is to help teams reach decisions faster:

- Which customer is blocked and why?
- Which account is at risk this week?
- Which workflow failed?
- Which deploy affected paying users?
- Which support tickets need engineering context?
- What should the team do next?

Think of CE as the open-source on-ramp. Think of WhoDB EE as the AI data platform for teams that want Palantir-like operational intelligence without enterprise lock-in.

---

## 🎮 Try WhoDB Now

<div align="center">

**Experience WhoDB in action without any setup**

<table>
<tr>
<td align="center" width="50%">
<h3>🌐 Live Demo</h3>
<p>Try WhoDB instantly with our sample database</p>
<a href="https://whodb.com/demo/login?host=quick-container-491288b0-3138-48fa-93b4-1e730296c0b7.hello.svc.cluster.local&username=user&password=password&database=Adventureworks">
<img src="./docs/images/01-login-page.png" alt="Login Page" width="80%"/>
</a>
<p><a href="https://whodb.com/demo/login?host=quick-container-491288b0-3138-48fa-93b4-1e730296c0b7.hello.svc.cluster.local&username=user&password=password&database=Adventureworks"><strong>Launch Demo →</strong></a></p>
<p><em>Pre-filled with sample PostgreSQL database</em></p>
</td>
<td align="center" width="50%">
<h3>🎥 Video Demo</h3>
<p>Watch WhoDB in action</p>
<a href="https://youtu.be/hnAQcYYzcLo">
<img src="https://img.youtube.com/vi/hnAQcYYzcLo/maxresdefault.jpg" alt="WhoDB Demo Video" width="80%"/>
</a>
<p><a href="https://youtu.be/hnAQcYYzcLo"><strong>Watch Video →</strong></a></p>
<p><em>Complete walkthrough of features</em></p>
</td>
</tr>
</table>

</div>

---

## 🚀 Quick Start

### Option 1: Docker (Recommended)

The fastest way to get started with WhoDB:

```bash
docker run -it -p 8080:8080 clidey/whodb
```

Then open [http://localhost:8080](http://localhost:8080) in your browser.

### Option 2: Docker Compose

For more control and configuration:

```yaml
version: "3.8"
services:
  whodb:
    image: clidey/whodb
    ports:
      - "8080:8080"
    environment:
      # AI Integration (Optional)
      # Ollama Configuration
      - WHODB_OLLAMA_HOST=localhost
      - WHODB_OLLAMA_PORT=11434

      # Anthropic Configuration
      - WHODB_ANTHROPIC_API_KEY=your_key_here
      # - WHODB_ANTHROPIC_ENDPOINT=https://api.anthropic.com/v1

      # OpenAI Configuration
      - WHODB_OPENAI_API_KEY=your_key_here
      # - WHODB_OPENAI_ENDPOINT=https://api.openai.com/v1

      # Generic AI Providers (OpenAI-compatible endpoints)
      # Use WHODB_AI_GENERIC_<ID>_* to add any OpenAI-compatible provider.
      # <ID> can be any unique identifier (e.g., LMSTUDIO, OPENROUTER).
      #
      # LM Studio example:
      # - WHODB_AI_GENERIC_LMSTUDIO_NAME=LM Studio
      # - WHODB_AI_GENERIC_LMSTUDIO_BASE_URL=http://host.docker.internal:1234/v1
      # - WHODB_AI_GENERIC_LMSTUDIO_MODELS=mistral-7b,llama-3-8b
      #
      # OpenRouter example:
      # - WHODB_AI_GENERIC_OPENROUTER_NAME=OpenRouter
      # - WHODB_AI_GENERIC_OPENROUTER_BASE_URL=https://openrouter.ai/api/v1
      # - WHODB_AI_GENERIC_OPENROUTER_API_KEY=your_key_here
      # - WHODB_AI_GENERIC_OPENROUTER_MODELS=google/gemini-2.0-flash-001,anthropic/claude-3.5-sonnet
      #
      # Requesty example:
      # - WHODB_AI_GENERIC_REQUESTY_NAME=Requesty
      # - WHODB_AI_GENERIC_REQUESTY_BASE_URL=https://router.requesty.ai/v1
      # - WHODB_AI_GENERIC_REQUESTY_API_KEY=your_key_here
      # - WHODB_AI_GENERIC_REQUESTY_MODELS=openai/gpt-4o-mini,anthropic/claude-3.5-sonnet
    # volumes: # (Optional for SQLite)
    #   - ./sample.db:/db/sample.db
```

### What's Next?

1. **Connect to a source** - Enter your database credentials on the login page
2. **Understand the shape** - Browse objects and visualize relationships
3. **Ask or query** - Use the Scratchpad, SQL, or AI chat to inspect live data
4. **Act with context** - Edit records, export results, seed test data, or move into the EE platform for governed workflows

📖 **For detailed installation options and configuration**, see our [Documentation](https://docs.whodb.com/)

---

## 💻 WhoDB CLI

WhoDB also offers a command-line interface with an interactive TUI (Terminal User Interface) and MCP server support for AI assistants. The CLI helps developers bring source context into local workflows, scripts, and AI tools.

### Features

- **Interactive TUI** - Full-featured terminal interface for source exploration
- **MCP Server** - Model Context Protocol support for Claude, Cursor, and other AI tools
- **Programmatic Access** - Query and export data from scripts
- **Cross-Platform** - Available for macOS, Linux, and Windows

### Quick Install

```bash
# macOS/Linux
curl -fsSL https://raw.githubusercontent.com/clidey/whodb/main/cli/install/install.sh | bash

# Homebrew (coming soon)
brew install whodb-cli

# npm
npm install -g @clidey/whodb-cli
```

### Usage

```bash
# Launch interactive TUI
whodb-cli

# Run as MCP server for AI assistants
whodb-cli mcp serve
```

📖 **For full CLI documentation**, see the [CLI README](./cli/README.md)

---

## 🛠️ Development Setup

### Prerequisites

- **GoLang** - Latest version recommended
- **PNPM** - For frontend package management
- **Node.js** - Version 16 or higher

### Editions

<table>
<tr>
<td width="50%">

**Community Edition (CE)**
- Open-source source exploration
- PostgreSQL
- MySQL / MariaDB
- SQLite3
- MongoDB
- Redis
- ElasticSearch
- AI chat and query workflows

</td>
<td width="50%">

**Enterprise Edition (EE) Platform**
- All CE databases
- Oracle
- SQL Server
- DynamoDB
- Athena
- Snowflake
- Cassandra
- More sources
- Governance, apps, and platform workflows

</td>
</tr>
</table>

📚 See [BUILD_AND_RUN.md](./BUILD_AND_RUN.md) for detailed build instructions and [ARCHITECTURE.md](./ARCHITECTURE.md) for architecture details.

### Frontend Development

Navigate to the `frontend/` directory and start the development server:

```bash
cd frontend
pnpm i
pnpm start
```

### Backend Development

#### 1. Build Frontend (First-Time Setup)

If the `core/build/` directory doesn't exist, build the frontend first:

```bash
cd frontend
pnpm install
pnpm run build
rm -rf ../core/build/
cp -r ./build ../core/
cd ..
```

> **Note:** This is only required once, as Go embeds the `build/` folder on startup.

#### 2. Setup AI Integration (Optional)

To enable natural language queries:

1. **Ollama** - Download from [ollama.com](https://ollama.com/)
   ```bash
   # Install Llama 3.1 8b model
   ollama pull llama3.1
   ```
   WhoDB will auto-detect installed models and show a **Chat** option in the sidebar.

2. **OpenAI/Anthropic** - Set environment variables (see Docker Compose example above)
3. **Any OpenAI-compatible provider** - Use `WHODB_AI_GENERIC_<ID>_*` environment variables to connect to LM Studio, OpenRouter, Requesty, or any provider with an OpenAI-compatible API (see Docker Compose example above)

#### 3. Start Backend Service

```bash
cd core
go run ./cmd/whodb
```

The backend will start on `http://localhost:8080`

---

## 💼 Use Cases

### 👨‍💻 For Developers

<table>
<tr>
<td width="50%">

**Local Development**
- Quick database inspection during development
- Debug production issues with read-only access
- Test API endpoints with real data
- Explore source and schema changes

</td>
<td width="50%">

**API Development**
- Validate data transformations
- Test query performance
- Generate mock data for testing
- Export data for integration tests

</td>
</tr>
</table>

### 📊 For Operators and Analysts

- Answer operational questions without waiting on a custom dashboard
- Export data to Excel, CSV, JSON, or SQL when needed
- Build complex filters visually
- Visualize relationships before making a decision

### 🧪 For QA Engineers

- Generate realistic test data
- Verify database state during testing
- Debug test failures quickly
- Validate data migrations

### 🛠️ For Database Administrators

- Monitor table structures and indexes
- Manage user data efficiently
- Quick schema exploration
- Emergency data fixes

### 🏢 For Teams Moving to the EE Platform

- Connect more source types as the company grows
- Govern who can see, query, change, and act on data
- Build shared ontology around business objects
- Give AI agents controlled source context
- Turn repeated data lookups into operational apps

---

## ❓ Frequently Asked Questions

<details>
<summary><strong>What makes WhoDB different from other database tools?</strong></summary>
<br>

WhoDB CE starts where database tools usually stop. It gives you fast source access, modern UX, visual schema exploration, and AI-assisted querying in a lightweight open-source package. The larger WhoDB EE platform builds on that source layer with more sources, governance, apps, and AI workflows for team decisions.

</details>

<details>
<summary><strong>Why is WhoDB open-core instead of only open source?</strong></summary>
<br>

We want the open-source parts of WhoDB to provide complete, useful value on their own. CE is not a toy demo. It is a real source exploration and query workspace developers can run today.

The full platform value comes when sources, ETLs, ontology, governance, apps, and the WhoDB AI Agent work together. That is where teams move from data access to faster decisions. As more parts become complete standalone value for the community, we plan to open-source more of them.

</details>

<details>
<summary><strong>Is WhoDB suitable for production use?</strong></summary>
<br>

Yes, WhoDB is production-ready and used by thousands of developers. For production environments, we recommend:
- Using read-only database accounts when possible
- Enabling SSL/TLS connections
- Consider Enterprise Edition for audit logging and advanced security features

</details>

<details>
<summary><strong>How does WhoDB handle large datasets?</strong></summary>
<br>

WhoDB implements several performance optimizations:
- Table virtualization for efficient rendering
- Lazy loading for large result sets
- Pagination controls
- Query result streaming

</details>

<details>
<summary><strong>Which databases are supported?</strong></summary>
<br>

**Community Edition:** PostgreSQL, MySQL, MariaDB, SQLite3, MongoDB, Redis, ElasticSearch

**Enterprise Edition Platform:** All CE databases plus Oracle, SQL Server, DynamoDB, Athena, Snowflake, Cassandra, and more source types for teams building governed operational workflows

</details>

<details>
<summary><strong>How do I deploy WhoDB?</strong></summary>
<br>

WhoDB can be deployed in multiple ways:
- **Docker** - Single command deployment
- **Docker Compose** - For production setups
- **Kubernetes** - For enterprise environments
- **Binary** - Direct installation on servers

See our [Quick Start](#quick-start) section for details.

</details>

<details>
<summary><strong>Does WhoDB store my credentials?</strong></summary>
<br>

No. WhoDB does not store database credentials by default. Connections are temporary and credentials are cleared when you close the browser. You can optionally configure connection profiles stored locally in your browser.

</details>

<details>
<summary><strong>Can I use WhoDB with AI features?</strong></summary>
<br>

Yes! WhoDB integrates with:
- **Ollama** - For local, private AI models
- **OpenAI** - GPT-4 and other OpenAI models
- **Anthropic** - Claude models
- **Any OpenAI-compatible provider** - LM Studio, OpenRouter, Requesty, vLLM, and more via `WHODB_AI_GENERIC_<ID>_*` environment variables

These integrations allow you to query your database using natural language instead of SQL. The EE platform expands the AI story toward governed source access, ontology, apps, and the WhoDB AI Agent for team workflows.

</details>

## 🤝 Contributing

We welcome contributions from the community! Whether it's bug reports, feature requests, or code contributions, we appreciate your help in making WhoDB better.

### How to Contribute

1. **Report Issues** - Found a bug? [Open an issue](https://github.com/clidey/whodb/issues)
2. **Request Features** - Have an idea? [Start a discussion](https://github.com/clidey/whodb/discussions)
3. **Submit PRs** - Want to contribute code? Check our [Contributing Guide](CONTRIBUTING.md)
4. **Improve Docs** - Help us improve documentation

### Development Resources

- [Contributing Guide](CONTRIBUTING.md) - Detailed contribution guidelines
- [Architecture](ARCHITECTURE.md) - Understanding the codebase
- [Build & Run](BUILD_AND_RUN.md) - Development setup instructions

---

## 📸 Screenshots

<details>
<summary><strong>View More Screenshots</strong></summary>

### Data Management
<img src="./docs/images/09-data-view-users-table.png" alt="Data View" width="100%"/>

### Add/Edit Records
<img src="./docs/images/11-data-view-add-row-dialog.png" alt="Add Row" width="100%"/>

### Advanced Filtering
<img src="./docs/images/16-data-view-where-conditions-popover.png" alt="Where Conditions" width="100%"/>

### Export Options
<img src="./docs/images/20-data-view-export-dialog.png" alt="Export Dialog" width="100%"/>

### Schema Graph Visualization
<img src="./docs/images/25-graph-view-with-controls.png" alt="Graph View" width="100%"/>

### Scratchpad Query Editor
<img src="./docs/images/28-scratchpad-code-editor.png" alt="Scratchpad" width="100%"/>

### Query Results
<img src="./docs/images/29-scratchpad-query-results.png" alt="Query Results" width="100%"/>

### Multiple Database Support
<img src="./docs/images/51-login-database-types-all-options.png" alt="Database Types" width="100%"/>

</details>

---

## 🏢 Infrastructure & Support

WhoDB's deployment and CI/CD are powered by [Clidey](https://clidey.com), a no-code DevOps platform.

<!-- **Build Status:** [![Build Status](https://hello.clidey.com/api/flows/status?id=b32257fa-1415-4847-a0f3-e684f5f76608&secret=cd74dbd5-36ec-42f9-b4f0-12ce9fcc762b)](https://clidey.com) -->

### Contact & Support

- **Email:** [support@clidey.com](mailto:support@clidey.com)
- **GitHub Issues:** [Report a bug](https://github.com/clidey/whodb/issues)
- **Discussions:** [Join the conversation](https://github.com/clidey/whodb/discussions)
- **Documentation:** [docs.whodb.com](https://docs.whodb.com/)

---

<div align="center">

### ⭐ Start With CE. Grow Into the Platform.

Run WhoDB CE when you need fast, open-source access to your operational data. Talk to us when your team needs the EE platform for more sources, governance, apps, and AI decisions.

[![GitHub stars](https://img.shields.io/github/stars/clidey/whodb?style=social)](https://github.com/clidey/whodb/stargazers)

---

**Built with ❤️ by the Clidey team**

*"Is it magic? Is it sorcery? No, it's just WhoDB!"*

</div>
