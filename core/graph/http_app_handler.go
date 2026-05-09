//go:build !arm && !riscv64

package graph

import "net/http"

var registeredAppGenerateHandler func(w http.ResponseWriter, r *http.Request)

// RegisterAppGenerateHandler registers the app generation stream handler.
func RegisterAppGenerateHandler(handler func(w http.ResponseWriter, r *http.Request)) {
	registeredAppGenerateHandler = handler
}

func appGenerateHandler(w http.ResponseWriter, r *http.Request) {
	if registeredAppGenerateHandler != nil {
		registeredAppGenerateHandler(w, r)
		return
	}
	http.Error(w, "App generation not available in this edition", http.StatusNotImplemented)
}
