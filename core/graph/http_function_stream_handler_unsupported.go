//go:build arm || riscv64

package graph

import "net/http"

// functionStreamHandler returns not-implemented on unsupported platforms.
func functionStreamHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Function streaming not available on this platform", http.StatusNotImplemented)
}
