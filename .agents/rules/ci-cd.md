---
paths:
  - ".github/**"
---

# CI/CD Rules

## Workflow Conventions
- Reusable workflows are prefixed with `_` (e.g., `_build-docker.yml`, `_deploy-apple.yml`); `release-ce.yml` orchestrates releases.
- Pin every action to a commit SHA. Add `step-security/harden-runner` to new jobs (`lint-check.yml` and `delete-draft-releases.yml` do not have it yet).
- Never hardcode or commit secrets. Workflows use GitHub environment-based secret access with minimal permissions per job.

## Releases
- A push to the `release` branch runs a production release with a minor version bump. A manual dispatch picks `stage-only` (edge/draft channels) or `production`, the bump (`major`/`minor`/`patch`/`current`, where `current` rebuilds without bumping) and the stores to publish.
- The `calculate-version` action derives the version from git tags plus the bump type. `publish-github-release` makes the drafted GitHub release public.

## Platform Build Tags
- Every CE-owned `//go:build !arm && !riscv64` file needs a matching `//go:build arm || riscv64` stub that defines the same symbols. See the `platform-constrained-handler` skill. Add-on HTTP routes register themselves with `graph.RegisterHTTPRoutes` instead of adding CE stubs.
- PR CI does not cross-compile. A missing stub first fails in the release job `Build Linux Server Binaries (CE)` (`_build-linux-terminal.yml`, `make server-all`). A local cross-build needs a CGO cross compiler such as `riscv64-linux-gnu-gcc` plus that workflow's BAML build-tag step, so at least check that every symbol in the tagged file also exists in its stub.

## Docker Images
- `clidey/whodb` (server) and `clidey/whodb-cli`, built for linux/amd64 and linux/arm64 on native runners.
