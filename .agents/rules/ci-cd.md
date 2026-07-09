---
paths:
  - ".github/**"
---

# CI/CD Rules

## Workflow Conventions
- Reusable workflows prefixed with `_` (e.g., `_build-docker.yml`, `_deploy-apple.yml`)
- Main orchestrators: `release-ce.yml` (CE), `release-ee.yml` (EE)
- All workflows use `step-security/harden-runner` and pinned action versions with SHA hashes

## Platform Build Tags
Every CE-owned `//go:build !arm && !riscv64` file MUST have a matching `_unsupported.go` stub. See `.agents/workflows/platform-constrained-handler.md`. Add-on HTTP routes should register themselves with `graph.RegisterHTTPRoutes` instead of adding CE stubs. If missed, the "Build Linux Binaries" job fails for RISC-V64.

## Version Bumping
Versions calculated by `calculate-version` action from git tags + bump type input (major/minor/patch/current).

## Key Secrets
Never hardcode or commit secrets. Workflows use GitHub environment-based secret access with minimal permissions per job.

## Docker Images
- CE: `clidey/whodb` (linux/amd64 + linux/arm64, native runners)
- EE: `clidey/whodb-ee`, `clidey/whodb-bridge`, `clidey/whodb-full`
- Bridge variant matrix is dynamic — adding a `DriverDefinition` in Java code auto-creates a new image variant
