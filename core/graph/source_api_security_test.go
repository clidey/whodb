package graph

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/clidey/whodb/core/src/common"
)

func TestConnectionAttemptsAreLimitedPerIPAndGlobally(t *testing.T) {
	connectionAttempts.Lock()
	previousStart := connectionAttempts.windowStart
	previousTotal := connectionAttempts.total
	previousByIP := connectionAttempts.byIP
	connectionAttempts.windowStart = time.Now()
	connectionAttempts.total = 0
	connectionAttempts.byIP = make(map[string]int)
	connectionAttempts.Unlock()
	t.Cleanup(func() {
		connectionAttempts.Lock()
		connectionAttempts.windowStart = previousStart
		connectionAttempts.total = previousTotal
		connectionAttempts.byIP = previousByIP
		connectionAttempts.Unlock()
	})

	request := httptest.NewRequest(http.MethodPost, "/api/query", nil)
	request.RemoteAddr = "192.0.2.10:1234"
	ctx := context.WithValue(context.Background(), common.RouterKey_Request, request)
	for range 30 {
		if !allowConnectionAttempt(ctx) {
			t.Fatal("connection attempt blocked before the per-IP limit")
		}
	}
	if allowConnectionAttempt(ctx) {
		t.Fatal("connection attempt exceeded the per-IP limit")
	}

	for i := 11; i < 16; i++ {
		request.RemoteAddr = "192.0.2." + strconv.Itoa(i) + ":1234"
		for range 30 {
			if !allowConnectionAttempt(ctx) {
				t.Fatal("connection attempt blocked before the global limit")
			}
		}
	}
	request.RemoteAddr = "192.0.2.20:1234"
	if allowConnectionAttempt(ctx) {
		t.Fatal("connection attempt exceeded the global limit")
	}

	connectionAttempts.Lock()
	connectionAttempts.windowStart = time.Now().Add(-time.Minute)
	connectionAttempts.Unlock()
	if !allowConnectionAttempt(ctx) {
		t.Fatal("connection attempt remained blocked after the window")
	}
}
