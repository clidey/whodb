---
paths:
  - "desktop-ce/**"
  - "desktop-common/**"
---

# Desktop App Rules

## Build System
- Uses Wails with the Go workspace file `go.work.desktop-ce`
- Frontend is embedded via `//go:embed all:frontend/dist/*`
- `make prepare` builds the frontend with `pnpm run $(FRONTEND_BUILD_CMD)` (`build:ce`) and copies `frontend/build/*` into `desktop-ce/frontend/dist` (build output); Wails then builds the native app

## Key Commands
```bash
cd desktop-ce && make dev          # Dev mode with hot reload
cd desktop-ce && make build        # Build for current platform
cd desktop-ce && make build-all    # All platforms
cd desktop-ce && make help         # All targets (packaging, signing, server binaries)
```

## Rules
- Desktop mode is detected at runtime with `env.GetIsDesktopMode()`; `WHODB_DESKTOP=true` is set automatically by the app
- Keyboard shortcuts that have Wails accelerators in `desktop-common/app.go` must be updated separately from frontend shortcuts
- Do not use build tags for edition control — edition is determined by entry point (`common.RunApp(common.RunConfig{Edition: ...})`)

## Verification
```bash
GOWORK=$PWD/go.work.desktop-ce go build -C desktop-ce -o test-build
rm -f desktop-ce/test-build
```
