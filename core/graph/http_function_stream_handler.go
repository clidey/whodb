//go:build !arm && !riscv64

package graph

import "net/http"

var registeredFunctionStreamHandler func(w http.ResponseWriter, r *http.Request)

// RegisterFunctionStreamHandler registers the function streaming handler.
func RegisterFunctionStreamHandler(handler func(w http.ResponseWriter, r *http.Request)) {
	registeredFunctionStreamHandler = handler
}

func functionStreamHandler(w http.ResponseWriter, r *http.Request) {
	if registeredFunctionStreamHandler != nil {
		registeredFunctionStreamHandler(w, r)
		return
	}
	http.Error(w, "Function streaming not available in this edition", http.StatusNotImplemented)
}
