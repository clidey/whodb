package mcp

import (
	"testing"

	"github.com/clidey/whodb/cli/internal/appcapture"
)

func TestAppCaptureChecksPassedIncludesBrowserAndTraceErrors(t *testing.T) {
	if !appCaptureChecksPassed(&appcapture.Result{}) {
		t.Fatal("clean capture failed")
	}
	for _, result := range []*appcapture.Result{
		{Diagnostics: appcapture.Diagnostics{ConsoleErrors: []string{"render failed"}}},
		{Diagnostics: appcapture.Diagnostics{PageErrors: []string{"page failed"}}},
		{Trace: []appcapture.TraceStep{{Kind: "handler-error"}}},
		{Trace: []appcapture.TraceStep{{Kind: "bridge-error"}}},
	} {
		if appCaptureChecksPassed(result) {
			t.Fatalf("capture with error passed: %#v", result)
		}
	}
}
