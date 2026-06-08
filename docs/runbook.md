# Runbook

This runbook collects the commands needed to develop, verify, package, and release DataFlow.

## Prerequisites

- Go 1.21+ for general development. The current `core/go.mod` toolchain may download a newer Go toolchain automatically.
- Node.js 22+.
- pnpm 10+.
- Docker with Buildx for runtime images.
- Helm for chart validation.
- Sealos CLI for local cluster-image packaging.

## Install Frontend Dependencies

```bash
cd dataflow
pnpm install --frozen-lockfile
```

If Corepack auto-adds a `packageManager` field during local commands, treat that as a local tooling side effect unless the project intentionally adopts it.

## Run Locally

Start the backend:

```bash
cd core
set -a
source .env.local
set +a
go run .
```

Start the frontend dev server:

```bash
cd dataflow
pnpm dev
```

The frontend dev server runs at `http://localhost:5173` and proxies backend API requests.

## Verification

Frontend:

```bash
cd dataflow
pnpm run typecheck
pnpm run build
pnpm run test
pnpm run lint
```

Backend build:

```bash
cd core
go build ./...
```

Backend tests:

```bash
cd core
go test ./...
```

Some backend packages load the BAML native library during tests. If the first run fails because the BAML download times out, retry after confirming the dylib exists in the local BAML cache, or set `BAML_LIBRARY_PATH` explicitly:

```bash
BAML_LIBRARY_PATH="$HOME/Library/Caches/baml/libs/0.218.1/libbaml_cffi-aarch64-apple-darwin.dylib" go test ./src/dashboard
```

Helm:

```bash
helm lint deploy/charts/dataflow
helm template dataflow deploy/charts/dataflow >/tmp/dataflow-helm-template.yaml
```

## Build Embedded Binary

```bash
cd dataflow
pnpm install --frozen-lockfile
pnpm run build

cd ..
rm -rf core/build
cp -R dataflow/build core/build

cd core
go build -tags prod -o dataflow-server .
```

Delete generated binaries after local testing unless they are intentional artifacts.

## Build Runtime Image

Build an `amd64` image by default for release-oriented checks:

```bash
docker buildx build \
  -f core/Dockerfile \
  --platform linux/amd64 \
  --build-arg VERSION=<version> \
  --build-arg TARGETARCH=amd64 \
  --build-arg PLATFORM=docker \
  -t dataflow-local:<version> \
  .
```

Run locally:

```bash
docker run --rm -p 8080:8080 dataflow-local:<version>
```

Open `http://localhost:8080`.

## Build Sealos Cluster Image Locally

See [deploy/README.md](../deploy/README.md) for the full packaging flow. The short version is:

1. Build and push a runtime image.
2. Update `deploy/charts/dataflow/values.yaml` image repository and tag.
3. Run `sealos registry save --registry-dir=registry_<arch> --arch <arch> .` from `deploy/`.
4. Build `deploy/Kubefile`.

## Release Preparation

Before pushing a new release tag:

1. Confirm the target branch is clean and synced with `origin/main`.
2. Run the frontend, backend, and Helm verification commands above.
3. Confirm GitHub Actions `Release` and `CodeQL` are green for the target commit.
4. Confirm the release-note comparison range. As of 2026-06-08, `v0.9.0` exists as a tag but does not have a GitHub Release.
5. Choose the next semantic version:
   - patch for fixes only.
   - minor when feature commits are included.
   - major only for breaking behavior or migration requirements.

Create and push the tag only after explicit release approval:

```bash
git tag vX.Y.Z
git push origin vX.Y.Z
```

The tag push triggers `.github/workflows/release.yaml`, which publishes runtime images, Sealos images, release tarballs, and the GitHub Release.

## Important Environment Variables

- `PORT`: backend HTTP port, default `8080`.
- `ENVIRONMENT=dev`: enables development-only GraphQL introspection/playground behavior.
- `WHODB_ALLOWED_ORIGINS`: comma-separated CORS origins.
- `WHODB_LOG_LEVEL`: log level.
- `WHODB_LOG_FORMAT`: use `json` for JSON logs.
- `WHODB_METADATA_DSN`: metadata database DSN.
- `WHODB_SESSION_DSN`: auth session DSN. Falls back to metadata DSN when unset.
- `WHODB_SESSION_ENCRYPTION_KEY`: server-side auth session encryption key.
- `WHODB_SESSION_TTL`: session lifetime, default `24h`.
- `WHODB_SEALOS_BOOTSTRAP_ENABLED`: set to `false` to disable Sealos bootstrap.
- `WHODB_STANDALONE_LOGIN_ENABLED`: set to `false` to disable standalone login.
- `WHODB_TOKENS`: enables API gateway mode when non-empty.
- `WHODB_OPENAI_API_KEY`, `WHODB_ANTHROPIC_API_KEY`, `WHODB_OLLAMA_HOST`, `WHODB_OLLAMA_PORT`: AI provider configuration.
- `WHODB_AI_GENERIC_<ID>_*`: generic AI provider configuration.
- `WHODB_ENABLE_AWS_PROVIDER`: enables AWS provider functionality.
- `BAML_LIBRARY_PATH`: explicit path to BAML native library for local macOS tests or bundled desktop builds.

## Troubleshooting

- Frontend `tsc: command not found`: run `pnpm install --frozen-lockfile` in `dataflow/`.
- Vite warns about large chunks: current bundle can exceed 1000 kB; treat as a performance follow-up unless release policy changes.
- Helm lint recommends `Chart.yaml` icon: informational only.
- `go test ./...` BAML failure on macOS: set `BAML_LIBRARY_PATH` after the native library is downloaded, then retry the affected package or full test suite.
