---
name: platform-constrained-handler
description: Add a new CE-owned HTTP handler with ARM/RISC-V64 build constraints (platform-excluded features)
---

# Add a Platform-Constrained HTTP Handler

Use this only when adding a CE-owned HTTP handler that depends on CGO libraries or platform-specific bindings unavailable on ARM/RISC-V64.

Do not use this workflow for edition or add-on HTTP routes. Add-on routes should be registered by the add-on package with `graph.RegisterHTTPRoutes`, so CE does not carry add-on route names or unsupported stubs.

## Steps

### 1. Create Main Handler
`core/graph/http_<name>_handler.go`:

```go
//go:build !arm && !riscv64

package graph

import "net/http"

func Setup<Name>Handler(mux *http.ServeMux) {
    mux.HandleFunc("/api/<name>", handle<Name>)
}

func handle<Name>(w http.ResponseWriter, r *http.Request) {
    // Implementation
}
```

### 2. Create Unsupported Stub
`core/graph/http_<name>_unsupported.go`:

```go
//go:build arm || riscv64

package graph

import "net/http"

func Setup<Name>Handler(mux *http.ServeMux) {
    mux.HandleFunc("/api/<name>", func(w http.ResponseWriter, r *http.Request) {
        http.Error(w, "not supported on this platform", http.StatusNotImplemented)
    })
}
```

### 3. Register in SetupHTTPServer
The handler registration call must be in unconditionally-compiled CE code (no build tags):
```go
Setup<Name>Handler(mux)
```

For add-on routes, register from the add-on package instead:

```go
func init() {
    graph.RegisterHTTPRoutes(func(router chi.Router) {
        router.Post("/api/<name>", handle<Name>)
    })
}
```

### 4. Verify Cross-Compilation
```bash
# Must pass — CI will fail if this breaks
GOOS=linux GOARCH=riscv64 go build ./graph/
GOOS=linux GOARCH=arm GOARM=7 go build ./graph/

# Normal build
cd core && go build ./cmd/whodb
```

## Key Rule
Every file with `//go:build !arm && !riscv64` MUST have a matching `_unsupported.go` stub that provides the same exported function signatures. If missed, CI fails on the "Build Linux Binaries" job.

## Existing Examples
- `http_ai_stream_handler.go` → `http_ai_stream_unsupported.go`
