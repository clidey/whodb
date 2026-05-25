---
name: desktop-feature
description: Add or modify desktop app functionality (Wails-based native apps)
---

# Desktop App Feature

## Architecture
```
desktop-common/         # Shared: App struct, window management, menus, RunApp()
desktop-ce/             # CE entry point, Makefile, wails.json
```

CE desktop calls `common.RunApp("ce", ...)`. EE is the same pattern in `ee/desktop/`.

## Adding Desktop-Specific Behavior

### 1. Runtime Detection
```go
import "github.com/clidey/whodb/core/src/env"

if env.IsDesktopApp() {
    // Desktop-specific behavior
}
```

### 2. Menu / Accelerator Changes
Edit `desktop-common/app.go`. Wails accelerators are separate from frontend keyboard shortcuts — both must be updated for shared shortcuts.

### 3. Window Management
All window logic lives in `desktop-common/app.go` (App struct methods).

## Build and Test

### Development
```bash
cd desktop-ce && make dev    # Hot reload mode
```

### Production Build
```bash
cd desktop-ce && make build          # Current platform
cd desktop-ce && make build-mac      # macOS universal binary
cd desktop-ce && make build-all      # All platforms
```

### Verification
```bash
# Must build with workspace
GOWORK=$PWD/go.work.desktop-ce go build -C desktop-ce -o test-build
rm -f desktop-ce/test-build

# Frontend must also build (it gets embedded)
cd frontend && pnpm run build:ce
```

## macOS Signing (release only)
```bash
make build-mac-signed      # Build + sign
make notarize              # Apple notarization
make build-mac-dmg         # DMG installer
```

Requires: `NOTARY_PROFILE`, `APPLE_ID`, `TEAM_ID`, `APPLE_APP_PASSWORD` env vars.

## Key Rules
- Entry points are thin — call `common.RunApp()`
- Edition not controlled by build tags, only by which entry point compiles
- `WHODB_DESKTOP=true` is set automatically at runtime
- Go workspace file: `go.work.desktop-ce`
