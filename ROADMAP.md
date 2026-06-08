# Roadmap

This roadmap is a planning guide for DataFlow release work. It should stay tied to current code and product direction rather than speculative platform ideas.

## Current Release Focus

- Stabilize MongoDB collection browsing and editing:
  - Collection Table View as the default browsing surface.
  - Preserved top-level document field order.
  - Inline scalar editing, focused JSON editing for complex fields, and whole-document replacement edits.
- Protect user work in the database workspace:
  - Unsaved database edit indicators on tabs.
  - Leave guard for tab close, browser refresh or close, and switching away from the database workspace.
- Keep release packaging reliable:
  - Runtime image build through `core/Dockerfile`.
  - Sealos cluster image packaging through `deploy/Kubefile` and `.github/workflows/release.yaml`.
  - Release assets for both `amd64` and `arm64` produced by GitHub Actions.

## Near-Term Priorities

1. Keep MongoDB data editing predictable and covered by unit tests.
2. Reduce frontend bundle size where it materially improves startup or editing workflows.
3. Improve release notes and tag discipline so public GitHub Releases match version tags.
4. Keep Helm values and Sealos install behavior aligned with runtime environment variables.
5. Expand documentation for architecture, runbook, and UI interaction contracts as release scope grows.

## Later Candidates

- Database expansion candidates are tracked in [docs/database-expansion-plan.md](docs/database-expansion-plan.md).
- Dashboard persistence and collaboration should be revisited only after core browsing and editing workflows are stable.
- Advanced import/export workflows should be prioritized based on user data-size and format feedback.

## Release Gate

Before a release tag is pushed:

- `cd dataflow && pnpm run typecheck && pnpm run build && pnpm run test && pnpm run lint`
- `cd core && go build ./...`
- `cd core && go test ./...` or a documented equivalent when native-library setup is required
- `helm lint deploy/charts/dataflow`
- `helm template dataflow deploy/charts/dataflow`
- Confirm GitHub Actions `Release` and `CodeQL` are green for the target commit.
- Confirm whether the release notes should be based on the latest version tag or the latest published GitHub Release.
