# References

This file records external and internal references that shape DataFlow implementation and release work.

## Internal Project References

- [README.md](../README.md): project overview, local development, embedded binary, Docker image, and Sealos packaging pointers.
- [AGENTS.md](../AGENTS.md): project rules for AI-assisted changes.
- [PRODUCT.md](../PRODUCT.md): product positioning, users, tone, and design principles.
- [CONTEXT.md](../CONTEXT.md): product language for MongoDB, workspace tabs, sidebar focus, and leave guard behavior.
- [ROADMAP.md](../ROADMAP.md): release focus and near-term priorities.
- [deploy/README.md](../deploy/README.md): Sealos packaging layout and local packaging flow.
- [docs/database-expansion-plan.md](database-expansion-plan.md): candidate database expansion priorities.
- [docs/adr/0001-bound-mongodb-field-inference-to-a-document-sample.md](adr/0001-bound-mongodb-field-inference-to-a-document-sample.md): MongoDB field inference decision.
- [docs/adr/0002-save-mongodb-json-document-edits-as-replacements.md](adr/0002-save-mongodb-json-document-edits-as-replacements.md): MongoDB JSON replacement edit decision.
- [dataflow/docs/semantic-test-contract.md](../dataflow/docs/semantic-test-contract.md): UI semantics contract for automation.

## Technology References

- React 19 and Vite power the frontend app under `dataflow/`.
- Tailwind CSS 4 tokens are defined in `dataflow/src/globals.css`.
- Zustand stores own frontend workspace and auth state.
- Apollo Client sends GraphQL requests to `/api/query`.
- gqlgen generates Go GraphQL server code under `core/graph`.
- Chi handles HTTP routing and middleware in `core/src/router`.
- GORM is the base for SQL-like database plugins.
- Helm and Sealos packaging live under `deploy/`.

## Release References

- GitHub Actions release workflow: [.github/workflows/release.yaml](../.github/workflows/release.yaml).
- PR packaging workflow: [.github/workflows/pr-docker-build.yml](../.github/workflows/pr-docker-build.yml).
- Runtime Dockerfile: [core/Dockerfile](../core/Dockerfile).
- Helm chart: [deploy/charts/dataflow](../deploy/charts/dataflow).

## Current Release Note

As of 2026-06-08, the repository has a `v0.9.0` tag but GitHub Releases only list `v0.1.0` as the latest published release. When preparing the next release, choose release-note scope explicitly:

- `v0.9.0..HEAD` for changes since the latest version tag.
- `v0.1.0..HEAD` for changes since the latest published GitHub Release.
