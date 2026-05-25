---
paths:
  - "desktop-ce/**"
  - "desktop-common/**"
---

# Desktop App Rules

## Build System
- Uses Wails with Go workspace files: `go.work.desktop-ce`
- Frontend is embedded via `//go:embed all:frontend/dist/*`
- Build flow: `make prepare` → builds frontend → copies to `desktop-*/frontend/dist/` → Wails builds native app

## Key Commands
```bash
cd desktop-ce && make dev          # Dev mode with hot reload
cd desktop-ce && make build        # Build for current platform
cd desktop-ce && make build-all    # All platforms
```

## Rules
- Desktop mode detected at runtime via `env.IsDesktopApp()`
- Keyboard shortcuts that have Wails accelerators in `desktop-common/app.go` must be updated separately from frontend shortcuts
- Do not use build tags for edition control — edition is determined by entry point
- `WHODB_DESKTOP=true` is set automatically at runtime

## Verification
```bash
GOWORK=$PWD/go.work.desktop-ce go build -C desktop-ce -o test-build
rm -f desktop-ce/test-build
```
