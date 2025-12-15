# Desktop App Development

WhoDB has native desktop apps built with [Wails](https://wails.io/). The desktop apps embed the frontend and run the backend locally.

## Directory Structure

```
desktop-ce/              # CE desktop app
  main.go               # Entry point - calls common.RunApp("ce", ...)
  Makefile              # CE-specific config, includes Makefile.common
  wails.json            # Wails config
  frontend/dist/        # Built frontend (copied during build)

ee/desktop/             # EE desktop app (same structure)

desktop-common/         # Shared desktop code
  app.go               # Wails App struct, window management, menus
  run.go               # RunApp() - starts Wails with embedded server
  Makefile.common      # Shared build targets
```

## Go Workspace Files

Desktop builds use separate workspace files to resolve dependencies correctly:
- `go.work.desktop-ce` - CE desktop workspace
- `ee/go.work.desktop` - EE desktop workspace

## Development Commands

```bash
# CE Desktop
cd desktop-ce && make dev          # Development mode with hot reload
cd desktop-ce && make build        # Build for current platform

# EE Desktop
cd ee/desktop && make dev
cd ee/desktop && make build

# Cross-platform builds (from desktop-ce or ee/desktop)
make build-windows    # Windows AMD64
make build-mac        # macOS (universal binary)
make build-linux      # Linux AMD64
make build-all        # All platforms
```

## Build Process

1. `make prepare` - Cleans artifacts, builds frontend via `pnpm run build:ce` (or `build:ee`)
2. Copies `frontend/build/*` to `desktop-*/frontend/dist/`
3. Wails embeds `frontend/dist` via `//go:embed all:frontend/dist/*`
4. Wails builds native app with embedded assets

## Key Environment Variables

```bash
ENVIRONMENT=dev           # Enables dev mode (set automatically by make dev)
WHODB_DESKTOP=true        # Set automatically by desktop apps at runtime
```

## Desktop-Specific Code Patterns

Desktop mode is detected at runtime:
```go
import "github.com/clidey/whodb/core/src/env"

if env.IsDesktopApp() {
    // Desktop-specific behavior
}
```

## Makefile Variables

Each edition's Makefile sets these before including `Makefile.common`:
```makefile
EDITION := ce                    # or "ee"
PKG_ID := com.clidey.whodb.ce   # Bundle ID
GOWORK := go.work.desktop-ce     # Workspace file
BUILD_TAGS :=                    # Empty for CE, "-tags ee" for EE
FRONTEND_BUILD_CMD := build:ce   # Frontend build script
APP_NAME := WhoDB.app            # macOS app name
```

## Signing & Notarization (macOS)

For release builds, set these environment variables:
```bash
NOTARY_PROFILE=your-profile
APPLE_ID=your@apple.id
TEAM_ID=XXXXXXXXXX
APPLE_APP_PASSWORD=xxxx-xxxx-xxxx-xxxx
```

Then run:
```bash
make build-mac-signed      # Build + sign
make notarize              # Notarize with Apple
make build-mac-dmg         # Create DMG installer
```
