# CI/CD and GitHub Actions

WhoDB uses GitHub Actions for automated builds, testing, and multi-platform deployment.

## Workflow Structure

```
.github/workflows/
  release-ce.yml          # Main CE release orchestrator
  _build-*.yml            # Reusable build workflows (called by release)
  _deploy-*.yml           # Reusable deploy workflows
  _sign-validate.yml      # Code signing validation
  _verify-deployment.yml  # Post-deployment verification
  claude.yml              # Claude Code integration
  claude-code-review.yml  # Automated PR review

.github/actions/
  calculate-version/      # Semantic version calculation
  deployment-summary/     # Release summary generator

.github/scripts/
  appstore-connect.sh     # Apple App Store Connect API
  build-appimage.sh       # AppImage packaging
  generate-homebrew-cask.sh  # Homebrew formula generation

.github/templates/
  RELEASE_TEMPLATE.md     # GitHub release notes template
  AppxManifest.xml.template  # Windows Store manifest
  homebrew-cask.rb.template  # Homebrew cask template
```

## Release Workflow (release-ce.yml)

The main release workflow orchestrates all builds and deployments.

### Triggers

- **Push to `release` branch** - Automatic deployment
- **Manual dispatch** - Fine-grained control

### Deployment Modes

| Mode | Description | Channels |
|------|-------------|----------|
| `stage-only` | Testing builds | Docker (tag), Snap (edge), MS Store (draft), Apple (TestFlight) |
| `production` | User releases | Docker (latest), Snap (stable), MS Store (prod), Apple (App Store) |

### Version Bumping

```
major    # 1.0.0 → 2.0.0
minor    # 0.61.0 → 0.62.0
patch    # 0.61.0 → 0.61.1
current  # Rebuild without version increment
```

### Store Selection

Each deployment target can be enabled individually:
- Docker Hub
- Snap Store (Linux)
- Microsoft Store (Windows)
- Apple App Store (macOS)
- Linux Terminal binaries (amd64, arm64, riscv64, armv6, armv7)

## Reusable Build Workflows

Prefixed with `_` to indicate they're called by other workflows.

### _build-docker.yml
- Builds Docker images for linux/amd64 and linux/arm64
- Uses native ARM runners for true ARM64 builds
- Outputs artifacts for multi-arch manifest

### _build-apple.yml
- Builds macOS DMG (direct download) and MAS (App Store) variants
- Universal binary (Intel + Apple Silicon)
- Handles code signing with Apple certificates

### _build-windows.yml
- Builds MSIX package for Microsoft Store
- Code signing with certificate
- Supports Windows AMD64

### _build-snap.yml
- Builds Snap package for Snapcraft/Ubuntu Store
- Supports multiple architectures

### _build-linux-terminal.yml
- Builds standalone terminal binaries (no GUI)
- Cross-compiles for: amd64, arm64, riscv64, armv6, armv7

## Deployment Workflows

### _deploy-docker.yml
- Pushes multi-arch manifest to Docker Hub
- Tags: `latest`, `vX.Y.Z`, `edge` (for staging)

### _deploy-apple.yml
- Uploads to App Store Connect
- Handles TestFlight (staging) vs App Store (production)
- Notarization for DMG downloads

### _deploy-microsoft.yml
- Submits to Microsoft Partner Center
- Draft submission (staging) vs production release

### _deploy-snap.yml
- Publishes to Snapcraft
- Edge channel (staging) vs stable (production)

### _deploy-homebrew.yml
- Updates Homebrew cask formula
- Generates SHA256 checksums

## Custom Actions

### calculate-version
Calculates semantic version based on:
- Git tags (latest tag as base)
- Bump type input (major/minor/patch/current)
- Outputs version for all downstream jobs

### deployment-summary
Generates deployment summary with:
- Build artifacts and sizes
- Store deployment status
- Download links

## Environment Secrets Required

```
# Docker
DOCKER_USERNAME
DOCKER_PASSWORD

# Apple
APPLE_DEVELOPER_CERTIFICATE_P12_BASE64
APPLE_DEVELOPER_CERTIFICATE_PASSWORD
APPLE_ID
APPLE_TEAM_ID
APPLE_APP_PASSWORD
ASC_ISSUER_ID
ASC_KEY_ID
ASC_KEY_BASE64

# Microsoft
AZURE_TENANT_ID
AZURE_CLIENT_ID
AZURE_CLIENT_SECRET
MS_STORE_SELLER_ID
MS_STORE_CLIENT_ID
MS_STORE_CLIENT_SECRET

# Snapcraft
SNAPCRAFT_STORE_CREDENTIALS

# Code Signing
WINDOWS_CERTIFICATE_P12_BASE64
WINDOWS_CERTIFICATE_PASSWORD
```

## Running Releases

### Stage Release (Testing)
1. Go to Actions → Release CE
2. Select `stage-only` mode
3. Choose stores to deploy
4. Run workflow

### Production Release
1. Merge to `release` branch, OR
2. Manual dispatch with `production` mode
3. Enable `publish-github-release` to make release public

## Security

All workflows use:
- `step-security/harden-runner` for egress control
- Pinned action versions with SHA hashes
- Environment-based secret access
- Minimal permissions per job
