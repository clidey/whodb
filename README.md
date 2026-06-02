## Overview

DataFlow is the web database workspace in this repository. It runs on top of the Go backend in [`core/`](./core) and is positioned as a Sealos-native database GUI plus a lightweight BI panel.

In the current version, the product is centered on three primary jobs:

- Browse and maintain database objects and records
- Write SQL, MongoDB, or Redis queries and commands
- Turn query results into charts and reusable dashboards

### Supported Databases

| Relational | Document / Key-Value | Analytics |
|------------|----------------------|-----------|
| PostgreSQL | MongoDB | ClickHouse |
| MySQL | Redis | |

### Core Workflows

- Browse databases, schemas, tables, collections, and keys
- View, edit, insert, delete, and export data
- Run SQL, Mongo shell-style, and Redis commands in editors
- Build charts and dashboard widgets from query results
- Work across PostgreSQL, MySQL, MongoDB, Redis, and ClickHouse in one workspace

## Local Development

### Prerequisites

- Go 1.21+
- Node.js 22+
- pnpm 10+

### Start the App

```bash
# Install frontend dependencies
cd dataflow
pnpm install

# Terminal 1: backend
cd core
set -a
source .env.local
set +a
go run .

# Terminal 2: frontend
cd ../dataflow
pnpm dev
```

The frontend dev server runs at `http://localhost:5173` and proxies API requests to the backend.

## Frontend Checks

```bash
cd dataflow
pnpm run typecheck
pnpm run build
pnpm run test
```

## Build an Embedded Binary

To build a production binary that embeds the frontend assets:

```bash
cd dataflow
pnpm install
pnpm run build

cd ..
rm -rf core/build
cp -R dataflow/build core/build

cd core
go build -tags prod -o dataflow-server .
```

## Build a Docker Image

```bash
docker build -f core/Dockerfile -t dataflow-local .
docker run --rm -p 8080:8080 dataflow-local
```

Open `http://localhost:8080` after the container starts.

## Build a Sealos Cluster Image

Sealos packaging files live under [`deploy/`](./deploy).

- PR workflow: [`/.github/workflows/pr-docker-build.yml`](./.github/workflows/pr-docker-build.yml)
- Release workflow: [`/.github/workflows/release.yaml`](./.github/workflows/release.yaml)
- Packaging details: [`deploy/README.md`](./deploy/README.md)
- User override template: [`deploy/charts/dataflow/dataflow-values.yaml`](./deploy/charts/dataflow/dataflow-values.yaml)

## Project Structure

```text
core/                   # Go backend
  server.go             # Entry point
  src/plugins/          # Database connectors
  graph/                # GraphQL schema and resolvers
  Dockerfile            # Production image build

dataflow/               # React 19 + TypeScript frontend
  src/main.tsx          # Entry point
  src/stores/           # Zustand stores
  src/components/       # Database, editor, analysis, and layout UI

deploy/                 # Sealos cluster image packaging

dev/                    # Local database fixtures and helper scripts
docs/                   # Product and engineering docs
```

## Tech Stack

- Backend: Go, GraphQL (gqlgen), Chi, GORM
- Frontend: React 19, TypeScript, Zustand, Apollo Client, Vite, Tailwind CSS 4, Monaco Editor, ECharts, react-grid-layout

## License

Apache 2.0. See [`LICENSE`](./LICENSE).
