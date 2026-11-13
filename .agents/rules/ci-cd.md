---
paths:
  - ".github/**"
---

# CI/CD Rules

## Workflow Conventions
- Reusable workflows are prefixed with `_` (e.g., `_build-docker.yml`, `_deploy-apple.yml`); `release-ce.yml` orchestrates releases.
- Pin every action to a commit SHA. Add `step-security/harden-runner` to new jobs (`lint-check.yml` and `delete-draft-releases.yml` do not have it yet).
- Never hardcode or commit secrets. Workflows use GitHub environment-based secret access with minimal permissions per job.
- PostHog source-map upload (`_build-frontend.yml`) needs the presigned S3 hosts `posthog-cloud-prod-us-east-1-app-assets.s3-accelerate.amazonaws.com` and `s3.us-east-1.amazonaws.com` in the harden-runner allowlist, not only the PostHog domains.

## Releases
- A push to the `release` branch runs a production release with a minor version bump. A manual dispatch picks `stage-only` (edge/draft channels) or `production`, the bump (`major`/`minor`/`patch`/`current`, where `current` rebuilds without bumping) and the stores to publish.
- The `calculate-version` action derives the version from git tags plus the bump type. `publish-github-release` makes the drafted GitHub release public.

## Mac App Store Certificates
- The Mac App Distribution and Mac Installer Distribution certs (team `F32XW59S3D`) last one year; the current pair expires 2027-09-29. Regenerate the provisioning profile on every renewal.
- CI reads `APPLE_MAS_CERTIFICATES_P12`/`_PASSWORD` (one p12 holding both identities), `APPLE_MAS_PROVISIONING_PROFILE` and `APPLE_MAS_CERT_INSTALLER`; the application identity is found in the imported keychain (`_build-apple.yml`, `_deploy-apple.yml`).
- If both certs share one CSR, a Keychain p12 export holds only one identity on the runner. Rebuild it with `openssl pkcs12 -export -legacy -inkey key.pem -in app.pem -certfile installer.pem`.
- Test a renewal with a `release-ce.yml` dispatch: `deployment-mode=stage-only`, `deploy-apple=true` (TestFlight only). The user runs it.

## Platform Build Tags
- Every CE-owned `//go:build !arm && !riscv64` file needs a matching `//go:build arm || riscv64` stub that defines the same symbols. See the `platform-constrained-handler` skill. Add-on HTTP routes register themselves with `graph.RegisterHTTPRoutes` instead of adding CE stubs.
- PR CI does not cross-compile. A missing stub first fails in the release job `Build Linux Server Binaries (CE)` (`_build-linux-terminal.yml`, `make server-all`). A local cross-build needs a CGO cross compiler such as `riscv64-linux-gnu-gcc` plus that workflow's BAML build-tag step, so at least check that every symbol in the tagged file also exists in its stub.

## Docker Images
- `clidey/whodb` (server) and `clidey/whodb-cli`, built for linux/amd64 and linux/arm64 on native runners.
